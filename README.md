# markcloud

A personal markdown hosting platform. Write locally, sync to your server, browse on the web at `markcloud.israelmanzi.com`.

## What it does

- Host markdown documents on your own domain
- Documents are private by default, with the option to make them public
- Browse and search your documents from the web
- Sync via a CLI that talks directly to your server
- Clean, minimalistic web interface — just your content

## How it works

Documents are authored locally as markdown files with YAML frontmatter for metadata (tags, visibility). The CLI manages a local content directory and syncs directly to the server over HTTP using SHA-256 diffing — only changed files are transferred. The server parses markdown, renders HTML, and stores everything in SQLite with full-text search.

## Infrastructure

```
  +---------+     HTTP API (manifest + upload)     +----------------+
  |  mc CLI  |------------------------------------->|  Server        |
  |          |     SHA-256 diffing, only sends      |  (Docker)      |
  |          |     changed files                    |                |
  +---------+                                      |  SQLite + FTS5 |
  local content_dir                                |  HTML templates |
  (git-tracked)                                    +-------+--------+
                                                           |
                                             markcloud.israelmanzi.com
```

## Tech stack

- **Language:** Go (CLI and server in one monorepo)
- **Storage:** SQLite with FTS5 for full-text search
- **Rendering:** goldmark (markdown to HTML, server-side)
- **Templates:** Go standard library html/template
- **Deployment:** Docker, GitHub Actions with automatic rollback

## CLI commands

- **sync** — sync content directory to server
- **upload** — upload a single markdown file with tags and visibility
- **ls** — list documents and directories
- **info** — view document metadata
- **public / private** — toggle visibility
- **rm** — delete a document
- **mv** — rename or move a document

## Configuration

The CLI reads from `~/.markcloud.yaml`:

```yaml
server_url: https://markcloud.israelmanzi.com
deploy_secret: your-secret
content_dir: ~/markdown
```

The server uses environment variables for secrets, passed via Docker.

## License

MIT
