# Docker 基础设施

本仓库不再维护本地构建的 MySQL/Redis 镜像与 docker-compose 编排。

这些基础设施服务由团队的公共 infra 平台统一提供（包括 MySQL、Redis 等）。

请在 infra 平台上创建或使用已有的服务实例，并在本项目的运行环境中通过环境变量或配置文件连接到这些外部服务。

快速指南

- 请向 infra 团队申请或确认 MySQL 与 Redis 服务的连接信息（主机、端口、用户名、密码等）。
- 将连接信息放入 `configs/env/config.dev.env` 或 `configs/env/config.prod.env`（或由环境注入）以供本项目使用。不要在仓库中保存明文密码。

示例（在运行时注入，切勿提交到仓库）

```bash
# MySQL
MYSQL_HOST=db.example.internal
MYSQL_PORT=3306
MYSQL_DATABASE=iam_contracts
MYSQL_USER=app_user
MYSQL_PASSWORD=<SECRET_FROM_VAULT>

# Redis
REDIS_HOST=redis.example.internal
REDIS_PORT=6379
REDIS_PASSWORD=<SECRET_FROM_VAULT>
```

其它说明

- 如果需要在本地进行集成测试，请使用 infra 团队提供的测试实例或在本地以临时方式使用外部容器（不要将其纳入仓库构建流程）。
- 所有与 infra 相关的敏感信息应通过 Vault、CI secrets 或环境变量注入。

如需帮助对接 infra，请联系平台/运维团队。
