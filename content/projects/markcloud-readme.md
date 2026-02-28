---
tags:
    - projects
    - misc
public: false
---

# markcloud

A personal markdown hosting platform. Write locally, push to GitHub, and your documents are rendered and served at `markcloud.israelmanzi.com`.

## What it does

- Host markdown documents on your own domain
- Documents are private by default, with the option to make them public
- Browse and search your documents from the web
- Upload via a CLI tool that syncs through GitHub
- Clean, minimalistic web interface — just your content

## How it works

Documents are authored locally as markdown files with frontmatter for metadata (tags, visibility). The CLI commits and pushes them to a private GitHub repository. A GitHub Action triggers on each push and syncs the content to your server. The server stores, indexes, and renders the documents as clean HTML pages.

## Infrastructure

```
                        git push
  +---------+    +-----------------+    GitHub Action     +----------------+
  |  mc CLI  |--->  GitHub Repo    |--------------------->|  Server        |
  |          |    |  (private)     |    sync changed      |  (Docker)      |
  |          |    |                |    files only         |                |
  |  ls/info |<---|  GitHub API    |                      |  SQLite + HTML |
  +---------+    +-----------------+                      |  templates     |
                                                          +-------+--------+
                                                                  |
                                                    markcloud.israelmanzi.com
                                                                  |
                                                  +---------------+---------------+
                                                  |                               |
                                            Authenticated                  Unauthenticated
                                            - Dashboard                   - Public docs index
                                            - All documents               - Public documents
                                            - Search                      - 404 for private
```

## Tech stack

- **Language:** Go (CLI and server)
- **Storage:** SQLite with FTS5 for full-text search
- **Rendering:** goldmark (markdown to HTML, server-side)
- **Templates:** Go standard library html/template
- **Deployment:** Docker on a personal server behind a reverse proxy

## CLI usage

The `mc` CLI manages your documents through GitHub:

- **upload** a markdown file to a directory with tags and visibility
- **ls** to list documents and directories
- **info** to view document metadata
- **public / private** to toggle visibility
- **rm** to delete a document
- **mv** to rename or move a document
- **status** to check the latest deployment
- **dry-run** on upload to generate a curl command instead of executing

## Configuration

The CLI reads from `~/.markcloud.yaml` for your GitHub repository and token. The server uses environment variables for secrets, passed via Docker.

## License

MIT
