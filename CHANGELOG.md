# Changelog

## 2026-03-01

### Direct CLI-to-server sync
- Replaced GitHub API sync with direct HTTP sync from CLI to server
- CLI now manages a local `content_dir` with git tracking for change history
- Removed `mc status` command (was GitHub Actions polling)
- Removed GitHub Action content sync job — content is synced by CLI
- Config changed from `github_repo`/`github_token` to `server_url`/`deploy_secret`/`content_dir`

### Markdown rendering
- Table of contents generation for documents with 2+ headings
- Callout/admonition blocks (`[!NOTE]`, `[!WARNING]`, `[!TIP]`, `[!IMPORTANT]`, `[!CAUTION]`)
- Wikilinks (`[[page]]` and `[[page|text]]`)
- Task list rendering
- Heading anchor IDs

### Backlinks
- Bidirectional link tracking between documents
- "Linked from" section on document pages

### SEO
- Sitemap at `/sitemap.xml`
- robots.txt at `/robots.txt`
- Canonical URLs on all pages
- OpenGraph tags (`og:title`, `og:description`, `og:type`, `og:url`, `og:site_name`, `og:image`)
- Twitter Card tags (`twitter:card`, `twitter:title`, `twitter:description`, `twitter:image`)
- JSON-LD structured data (Article schema) on document pages
- Meta description on document pages
- Default branded OG image for link previews

### Web UI
- Breadcrumb navigation
- Tag filtering (`/?tag=X`)
- RSS feed with autodiscovery at `/feed.xml`

### Documentation
- Added USAGE.md with full CLI reference
- Updated README.md for current architecture

## 2026-02-28

### Initial release
- Go monorepo with CLI (`mc`) and server
- CLI commands: upload, ls, info, public, private, rm, mv, status, sync
- SQLite with FTS5 for full-text search
- Markdown rendering with syntax highlighting (goldmark)
- Server-rendered HTML with Go templates
- Dark mode with theme toggle
- Mobile responsive design
- GitHub Action for content sync and app deployment
- Docker deployment with automatic rollback
- `.md` extension stripping from internal links
