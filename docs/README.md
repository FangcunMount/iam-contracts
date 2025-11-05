# IAM Contracts · 文档金字塔

欢迎来到 IAM Contracts 文档中心。本仓库围绕“金字塔”结构组织资料：**从全局概览 → 模块细节 → 运维与质量 → FAQ**，帮助不同角色快速定位信息。

```
概览（Overview）
├─ 模块（Modules）
├─ 运维 / 质量（Operations & Quality）
└─ FAQ / Troubleshooting
```

---

## 🧭 顶层 · 概览

- [系统总览](./overview/system-overview.md)：架构、技术栈、模块关系、一览图。
- [概念与术语](./overview/concepts-glossary.md)：DDD、CQRS、多租户、身份模型等关键术语。

***新同学建议路径***

1. 阅读系统总览把握全局；
2. 查阅概念词汇补齐背景；
3. 根据职责深入相应模块或运维/测试手册。

---

## 🧱 中层 · 模块文档

| 模块 | 说明 | 入口 |
|------|------|------|
| 👥 用户中心 (UC) | 用户、儿童档案、监护关系 | `./modules/uc/README.md` |
| 🔐 认证中心 (AuthN) | 登录策略、Token、JWKS | `./modules/authn/README.md` |
| 🛡️ 授权中心 (AuthZ) | 资源/角色策略、Casbin | `./modules/authz/README.md` |

模块目录内统一结构（示例）：

- `README.md`：模块概览、职责、架构图
- `ARCHITECTURE.md` / `DOMAIN_MODELS.md` / `DATA_MODELS.md`
- 业务流程或集成说明（视模块补充）
- API 细节以 **OpenAPI / Proto** 作为权威来源，避免重复维护

---

## 🔧 基座 · 运维与质量

- **Operations** → `./ops/`  
  部署（`deployment/`）、脚本、迁移策略、镜像仓库、权限配置等。

- **Quality & Testing** → `./quality/`  
  测试策略、API 测试指南、命令速查、后续质量门禁策略。

> 建议在 PR 中同步更新对应目录，保持流程与文档一致。

---

## ❓ 底座 · FAQ / Troubleshooting

- `./faq/README.md`  
  收敛常见问题、排障手册、操作提示。补充时请附带场景、影响与解决步骤，并回链至模块/运维文档。

---

## 🤝 贡献者须知

1. Fork → Branch (`feature/**`) → Commit → PR。
2. 更新代码同时照顾相关文档；若属新专题，请规划在合适的层级目录。
3. 使用 Markdown，避免重复描述。从源文件（OpenAPI、脚本）引用或链接，减少漂移。

更多项目信息：

- 仓库：<https://github.com/FangcunMount/iam-contracts>
- 问题反馈：<https://github.com/FangcunMount/iam-contracts/issues>
- 许可证：MIT（见 [LICENSE](../LICENSE)）

---

欢迎大家持续丰富文档，让知识沉淀更高效。 🎉
