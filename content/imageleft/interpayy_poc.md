---
tags:
    - projects
    - imageleft
    - interpayy
public: false
---

# INTERPAYY

Payment orchestration layer for Rwanda. Users pay and receive RWF via mobile money — stablecoins handle settlement behind the scenes, invisible to the end user.

INTERPAYY sits between consumers and payment providers as a plug-and-play middleware. Clients integrate through a single API. Providers are swapped or added without changing business logic. The core manages user wallets, exchange rates, and transaction lifecycles through a double-entry ledger.

## MVP Scope

- Fund-in via mobile money (MTN MoMo)
- Fund-out via mobile money
- Wallet-to-wallet transfers
- FX rate quoting
- Transaction history
- Single ramp provider integration

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full system design — modules, data flows, and abstraction boundaries.
