# AstianGO Hub

AstianGO Hub is a distributed platform for operating web crawlers, schedules, workers, task results, notifications, and data integrations from one control plane.

> This project is an independent fork derived from the open-source Crawlab project. AstianGO Hub is not affiliated with or endorsed by the Crawlab project or its maintainers. See [NOTICE.md](NOTICE.md) and [LICENSE](LICENSE) for attribution and license terms.

## Quick start

```yaml
services:
  master:
    image: goastian/astiango-hub:latest
    container_name: astiango_hub_master
    environment:
      ASTIANGO_NODE_MASTER: "Y"
      ASTIANGO_MONGO_HOST: "mongo"
    ports:
      - "8080:8080"
    depends_on:
      - mongo

  mongo:
    image: mongo:5
```

```bash
docker compose up -d
```

Open `http://localhost:8080`. The inherited development default is `admin/admin`; change it immediately and do not expose the service publicly until the security baseline in the modernization plan is complete.

## Canonical identifiers

| Purpose | Identifier |
| --- | --- |
| Product | `AstianGO Hub` |
| Repository | `github.com/goastian/astiango-hub` |
| Docker image | `goastian/astiango-hub` |
| Configuration prefix | `ASTIANGO_` |
| CLI and server binary | `astiango-hub` / `astiango-hub-server` |
| MCP package | `@goastian/astiango-hub-mcp` |
| Frontend package | `astiango-hub-ui` |

## Development

The repository is a Go workspace with the `backend`, `core`, `grpc`, `trace`, and `vcs` modules, plus a Vue frontend and TypeScript MCP server.

```bash
go test ./core/... ./grpc/... ./trace/... ./vcs/...
cd frontend/astiango-hub-ui && pnpm install && pnpm run build
cd ../../../mcp && pnpm install && pnpm run build
```

The complete enterprise hardening and modernization roadmap is in [docs/plan-modernizacion-servicio-empresarial.md](docs/plan-modernizacion-servicio-empresarial.md). The naming contract and remaining upstream compatibility dependencies are documented in [docs/rebranding-astiango-hub.md](docs/rebranding-astiango-hub.md).

## License and responsible use

Keep the original copyright and license notices when redistributing source or binaries. Operators are responsible for complying with applicable law, target-site terms, privacy requirements, rate limits, and `robots.txt` where applicable. See [DISCLAIMER.md](DISCLAIMER.md).
