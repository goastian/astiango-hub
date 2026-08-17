# AstianGO Hub

AstianGO Hub 是一个分布式网络爬虫管理平台，可统一管理节点、爬虫、任务、定时计划、结果、通知和数据集成。

> 本项目是基于开源 Crawlab 项目的独立分支，与 Crawlab 项目及其维护者不存在隶属或背书关系。版权归属和许可条款请参阅 [NOTICE.md](NOTICE.md) 与 [LICENSE](LICENSE)。

## 快速开始

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

访问 `http://localhost:8080`。继承的开发默认账号为 `admin/admin`；请立即修改密码，并在完成安全基线前避免将服务暴露到公网。

## 标准命名

| 用途 | 标识符 |
| --- | --- |
| 产品 | `AstianGO Hub` |
| 仓库 | `github.com/goastian/astiango-hub` |
| Docker 镜像 | `goastian/astiango-hub` |
| 环境变量前缀 | `ASTIANGO_` |
| CLI 与服务端程序 | `astiango-hub` / `astiango-hub-server` |
| MCP 包 | `@goastian/astiango-hub-mcp` |
| 前端包 | `astiango-hub-ui` |

企业级加固与现代化计划见 [docs/plan-modernizacion-servicio-empresarial.md](docs/plan-modernizacion-servicio-empresarial.md)，命名约定和上游兼容依赖见 [docs/rebranding-astiango-hub.md](docs/rebranding-astiango-hub.md)。
