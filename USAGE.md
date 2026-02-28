# markcloud

Personal markdown hosting platform. Write locally, sync to your server, browse on the web.

## Architecture

Go monorepo producing two binaries:

- **`mc`** — CLI for managing documents locally and syncing to the server
- **`server`** — Web server with SQLite storage, full-text search, and RSS

```
local markdown files
        │
    mc sync (HTTP API, SHA-256 diffing)
        │
   ┌────▼────┐
   │  server  │──── SQLite + FTS5
   └────┬────┘
        │
   web UI at your domain
```

The CLI manages a local `content_dir` (tracked with a local git repo for change history). On `mc sync`, it computes SHA-256 hashes, sends a manifest to the server to identify changes, then uploads only the files that differ. The server parses markdown, extracts metadata, renders HTML, and stores everything in SQLite.

Deployment: Docker container on a server, GitHub Actions workflow rebuilds and deploys on push to `main` with automatic rollback on health check failure.

## Build

Requires Go with CGO enabled (for SQLite FTS5).

```sh
# CLI
go build -tags fts5 -o mc ./cmd/mc

# Server
go build -tags fts5 -o server ./cmd/server
```

## Configuration

### CLI (`~/.markcloud.yaml`)

```yaml
server_url: https://markdown.example.com
deploy_secret: your-deploy-secret
content_dir: ~/markdown
```

`content_dir` is where your markdown files live. The CLI tracks changes with a local git repo inside this directory.

### Server

Configured via environment variables (typically in `.env`):

- `DEPLOY_SECRET` — shared secret for API authentication
- `DB_PATH` — path to SQLite database file

## CLI Commands

### `mc sync`

Sync all markdown files from `content_dir` to the server. Only uploads files that have changed (compared by SHA-256).

```sh
mc sync
```

### `mc upload`

Upload a single markdown file.

```sh
mc upload --path ./post.md --name my-post --dir blog --public --tags "go,web"
```

Flags:

- `--path` — path to the markdown file (required)
- `--name` — document name (defaults to filename without `.md`)
- `--dir` — target directory on the server
- `--tags` — comma-separated tags
- `--public` — make the document publicly visible

### `mc ls [dir]`

List documents in your content directory.

```sh
mc ls
mc ls blog
```

### `mc info <path>`

Show document metadata (public/private, tags).

```sh
mc info blog/my-post
```

### `mc public <path>` / `mc private <path>`

Toggle document visibility.

```sh
mc public blog/my-post
mc private blog/my-post
```

### `mc rm <path>`

Delete a document and sync the deletion.

```sh
mc rm blog/old-post
```

### `mc mv <source> <dest>`

Move or rename a document.

```sh
mc mv blog/draft notes/finished
```

## Frontmatter

Documents support YAML frontmatter for metadata:

```markdown
---
tags:
  - go
  - web
public: true
---

# My Document

Content here...
```

## Web Features

- Full-text search (SQLite FTS5)
- Table of contents generation
- Backlinks between documents
- Breadcrumb navigation
- Tag filtering
- GitHub-flavored markdown (tables, task lists, strikethrough)
- Syntax-highlighted code blocks
- Callout blocks (`[!NOTE]`, `[!WARNING]`, `[!TIP]`, `[!IMPORTANT]`, `[!CAUTION]`)
- Wikilinks (`[[page]]` and `[[page|text]]`)
- RSS feed at `/feed.xml`
- Sitemap at `/sitemap.xml`
- SEO: OpenGraph, Twitter Cards, JSON-LD, canonical URLs
- Light/dark theme toggle
