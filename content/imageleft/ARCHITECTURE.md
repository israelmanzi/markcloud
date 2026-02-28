---
tags:
    - projects
    - imageleft
    - interpayy
public: false
---

# INTERPAYY — Architecture

Payment orchestration microservice for Rwanda. Accepts local fiat (RWF) via mobile money, uses stablecoins as an internal settlement rail, and manages virtual user wallets. Users never interact with stablecoins directly — they pay and receive RWF.

## Design Principles

- **Provider-agnostic**: Ramp providers sit behind a common interface. Swappable without touching business logic.
- **External treasury**: Stablecoins are held by the ramp provider, not by INTERPAYY. We maintain a virtual ledger only.
- **Double-entry ledger**: Every money movement is a debit + credit pair. Balances are derived, never stored as a mutable field.
- **Stablecoin as settlement rail**: Crypto is an internal implementation detail, invisible to users.

## The Abstraction Layer

INTERPAYY is a plug-and-play middleware. Both sides — consumers and providers — connect through stable interfaces. The core business logic in the middle never changes regardless of what's plugged in at either end.

```
    CONSUMER SIDE                    INTERPAYY CORE                     PROVIDER SIDE
    (plug & play)                                                       (plug & play)

 ┌─────────────────┐                                                ┌──────────────────┐
 │   Mobile App    │─┐                                           ┌──│  Ramp Provider A  │
 ├─────────────────┤  │          ┌─────────────────────┐         │  ├──────────────────┤
 │   Web Client    │──┤          │                     │         ├──│  Ramp Provider B  │
 ├─────────────────┤  ├── API ──►│   Auth │ Txn │ FX   │── IF ──┤  ├──────────────────┤
 │  Third-Party    │──┤          │        │     │      │         ├──│  MoMo Direct     │
 │  Integration    │  │          │     Wallet Ledger   │         │  ├──────────────────┤
 ├─────────────────┤  │          │                     │         └──│  Future Provider  │
 │  Future Client  │─┘          └─────────────────────┘            └──────────────────┘
 └─────────────────┘

          API = REST API (public contract)          IF = Provider Interface (internal contract)
```

**Consumer side**: Any client that speaks HTTP can integrate. Mobile apps, web dashboards, third-party systems. The REST API is the stable public contract — clients are added or swapped without changing INTERPAYY internals.

**Provider side**: Any payment or ramp provider that implements the Provider Interface can be plugged in. Provider-specific logic (auth, request formatting, webhook parsing) is encapsulated in its own adapter. Providers are added or swapped without changing INTERPAYY internals.

**The core never knows** who is calling it or which provider will fulfill the request. It only knows its own domain: users, wallets, rates, and transaction lifecycles.

## Modules

### Auth

Handles user identity. Registration, login, session tokens, role-based access (user / admin). Every other module depends on Auth to identify who is making a request.

### Wallet

The internal ledger. Each user has a virtual wallet denominated in USD. All balance changes happen through double-entry journal entries — no direct balance mutations. The Wallet module exposes operations like credit, debit, and hold (for pending transactions). It knows nothing about how money enters or leaves the system.

### FX

Responsible for exchange rates between RWF and USD. Fetches rates from external sources, caches them, and provides time-limited quotes. When a transaction begins, FX locks a rate and attaches it to that transaction for auditability. Other modules call FX whenever they need to convert between currencies.

### Transaction

The orchestrator. Manages the lifecycle of on-ramp (fund-in), off-ramp (fund-out), and wallet-to-wallet transfer flows. Owns the state machine for each transaction (initiated, quoted, confirmed, processing, completed, failed). Coordinates between FX (for rate quotes), Wallet (for balance changes), and the Ramp Provider (for external money movement). Also handles webhook ingestion from providers and reconciliation.

### Ramp Provider

An abstraction layer over external payment/crypto providers. Defines a common interface that any provider must implement: quoting, on-ramp execution, off-ramp execution, status checks, and webhook handling. Provider-specific logic (API auth, request formatting, response parsing) lives inside individual provider implementations. The Transaction module talks only to this interface, never to a specific provider directly.

## How Modules Connect

```
                    ┌──────────────┐
                    │     Auth     │
                    └──────┬───────┘
                           │ identity context
                           ▼
┌────────┐  rate   ┌──────────────┐  balance ops   ┌──────────┐
│   FX   │◄────────│ Transaction  │───────────────►│  Wallet  │
└────────┘  quotes │  (orchestr.) │                └──────────┘
                   └──────┬───────┘
                          │ ramp calls + webhooks
                          ▼
                   ┌──────────────┐
                   │    Ramp      │
                   │   Provider   │
                   │  Interface   │
                   └──────┬───────┘
                          │
              ┌───────────┼───────────┐
              ▼           ▼           ▼
         Provider A  Provider B  Provider ...
```

- **Auth** sits upstream of everything. All requests pass through it.
- **Transaction** is the central coordinator. It never moves money or fetches rates itself — it delegates to Wallet and FX.
- **Wallet** is purely internal. It has no knowledge of providers or exchange rates.
- **FX** is a standalone service. It provides quotes on demand and locks rates when asked.
- **Ramp Provider** is the boundary between INTERPAYY and the outside world. Transaction talks to the interface; the interface dispatches to the correct provider implementation.

## Data Flows

### Fund-In (On-Ramp)

```
 User                    INTERPAYY                              Provider
  │                          │                                      │
  │──── deposit request ────►│                                      │
  │                          │                                      │
  │                          │── get rate ──► FX                    │
  │                          │◄── RWF/USD ──┘                      │
  │                          │                                      │
  │◄── confirm quote ────────│                                      │
  │──── user confirms ──────►│                                      │
  │                          │                                      │
  │                          │──── collect RWF via MoMo ──────────►│
  │                          │          (provider handles fiat      │
  │                          │           collection + stablecoin    │
  │  ┌───────────────────────│           conversion externally)     │
  │  │                       │                                      │
  │  │                       │◄──────── webhook: success ──────────│
  │  │                       │                                      │
  │  │                       │── credit ──► Wallet                  │
  │  │                       │                                      │
  │◄─┘  deposit confirmed   │                                      │
  │                          │                                      │
```

User deposits RWF. INTERPAYY quotes a rate, provider collects payment and handles stablecoin conversion. On webhook confirmation, the user's wallet is credited in USD.

### Fund-Out (Off-Ramp)

```
 User                    INTERPAYY                              Provider
  │                          │                                      │
  │── withdrawal request ───►│                                      │
  │                          │                                      │
  │                          │── get rate ──► FX                    │
  │                          │◄── USD/RWF ──┘                      │
  │                          │                                      │
  │◄── confirm quote ────────│                                      │
  │──── user confirms ──────►│                                      │
  │                          │                                      │
  │                          │── hold ──► Wallet                    │
  │                          │     (reserve funds, pending state)   │
  │                          │                                      │
  │                          │──── pay out RWF to MoMo ──────────►│
  │                          │          (provider converts          │
  │                          │           stablecoin to fiat         │
  │  ┌───────────────────────│           and sends to user)         │
  │  │                       │                                      │
  │  │                       │◄──────── webhook: success ──────────│
  │  │                       │                                      │
  │  │                       │── finalize debit ──► Wallet          │
  │  │                       │                                      │
  │◄─┘  RWF received on MoMo│                                      │
  │                          │                                      │
```

User requests RWF withdrawal. INTERPAYY holds funds in the wallet, provider handles stablecoin-to-fiat conversion and MoMo payout. On webhook confirmation, the hold is finalized as a debit.

### Wallet-to-Wallet Transfer

```
 Sender                  INTERPAYY                          Receiver
  │                          │                                  │
  │──── send request ───────►│                                  │
  │                          │                                  │
  │                          │── debit ──► Sender Wallet        │
  │                          │── credit ──► Receiver Wallet     │
  │                          │                                  │
  │◄── transfer complete ────│──── transfer received ──────────►│
  │                          │                                  │
```

Pure internal ledger operation. No provider, no FX. Instant.

## Boundaries

### What INTERPAYY Does Not Do

- Hold actual stablecoins or manage blockchain wallets
- Replace or compete with mobile money operators
- Act as a licensed financial institution (operates under provider licenses / regulatory sandbox)

### Future Roadmap

- Card payment support as an additional consumer-side payment method
- Multi-country expansion beyond Rwanda
- Cross-border transfers between users in different markets
- Multiple simultaneous providers with intelligent routing
- CBDC integration as a settlement rail option
