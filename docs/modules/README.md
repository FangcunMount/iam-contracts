# Modules

> 各业务域（AuthN / AuthZ / UC）的详细文档入口。

- `authn/`：认证中心 —— 登录策略、Token 服务、JWKS 管理。
- `authz/`：授权中心 —— 策略定义、Casbin 集成、资源模型。
- `uc/`：用户中心 —— 用户档案、儿童/监护关系、领域事件。

每个模块目录均遵循统一约定：
1. `README.md`：职责、架构概览、上下游关系。
2. `ARCHITECTURE.md` / `DOMAIN_MODELS.md` / `DATA_MODELS.md`：核心设计。
3. 业务流程、接口集成等专题文档（视模块补充），与代码/接口规范保持同步。
