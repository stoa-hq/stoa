# Stoa

[![Website](https://img.shields.io/badge/website-stoahq.eu-blue)](https://stoahq.eu)
[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-ffdd00?logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/stoahq)
[![Matrix](https://img.shields.io/matrix/stoa-dev?server_fqdn=matrix.pineconeops.com&fetchMode=guest)](https://matrix.to/#/#stoa-dev:matrix.pineconeops.com)

A lightweight, open-source headless, agentic commerce platform built with Go. Ships as a single binary with the admin panel and storefront embedded.

## Features

- **Headless Architecture** -- REST API (JSON)
- **Single Binary** -- Go backend with embedded SvelteKit frontends (Admin + Storefront)
- **MCP Servers** -- AI agents can shop in and manage the store via the Model Context Protocol
- **Plugin System** -- Extensible via hooks and custom API endpoints
- **Multi-language** -- Translation tables with locale-based API
- **Property Groups & Variants** -- Color, size, etc. with automatic combination generation
- **Full-text Search** -- PostgreSQL-based
- **RBAC** -- Role-based access control with granular API key permissions

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Docker + Docker Compose | latest | Database (and optional app container) |
| Go | 1.23+ | Build backend (local development only) |
| Node.js | 20+ | Build frontends (local development only) |
| PostgreSQL | 16+ | Database (provided via Docker) |

## Quick Start

```bash
git clone https://github.com/stoa-hq/stoa.git && cd stoa
cp config.example.yaml config.yaml
docker compose up -d
docker compose exec stoa ./stoa migrate up
docker compose exec stoa ./stoa admin create --email admin@example.com --password your-password
```

| What | URL |
|------|-----|
| Storefront | http://localhost:8080 |
| Admin Panel | http://localhost:8080/admin |
| API | http://localhost:8080/api/v1/health |

## Documentation

Full documentation is available in the **[StoA Docs](https://stoa-hq.github.io/docs/)**:

- [Introduction](https://stoa-hq.github.io/docs/guide/introduction) -- what Stoa is and why it exists
- [Quick Start](https://stoa-hq.github.io/docs/guide/quick-start) -- get up and running in minutes
- [Configuration](https://stoa-hq.github.io/docs/guide/configuration) -- all config options explained
- [API Overview](https://stoa-hq.github.io/docs/api/overview) -- authentication, endpoints, and usage
- [MCP Servers](https://stoa-hq.github.io/docs/mcp/overview) -- AI agent integration
- [Plugin System](https://stoa-hq.github.io/docs/plugins/overview) -- extend Stoa without forking
- [Payment Integration](https://stoa-hq.github.io/docs/plugins/payment) -- integrate any PSP

## License

Apache 2.0 -- see [LICENSE](LICENSE).
