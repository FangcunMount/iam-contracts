# 业务域

本文回答：`iam-contracts` 在新文档框架里，业务域层应该怎么组织，当前哪些正文已经迁入这一层，以及旧 `认证域 / 授权域 / 用户域` 目录现在该怎样看待。

## 30 秒结论

- `docs/02-业务域/` 现在开始承接**现状版业务域正文**，它是新框架里的业务能力层，而不是旧目录的索引页。
- 当前已经迁入的是三组核心正文：
  - [01-authn-认证、Token、JWKS.md](./01-authn-认证、Token、JWKS.md)
  - [02-authz-角色、策略、资源、Assignment.md](./02-authz-角色、策略、资源、Assignment.md)
  - [03-user-用户、儿童、Guardianship.md](./03-user-用户、儿童、Guardianship.md)
- 当前也已经补了一篇补充能力正文：
  - [04-suggest-儿童联想搜索.md](./04-suggest-儿童联想搜索.md)
- 旧 `docs/01-认证域/`、`docs/02-授权域/`、`docs/03-用户域/` 已归档到 [../_archive/](../_archive/README.md)。
- 这层文档的目标是“当前代码的可信镜像”，不是未来设计稿；结论优先回链到源码、合同和现有专题。

## 重点速查

| 想回答的问题 | 先打开哪里 |
| ---- | ---- |
| 当前认证域真正落地了什么？ | [01-authn-认证、Token、JWKS.md](./01-authn-认证、Token、JWKS.md) |
| 当前授权域真正落地了什么？ | [02-authz-角色、策略、资源、Assignment.md](./02-authz-角色、策略、资源、Assignment.md) |
| 当前 suggest 联想搜索到底怎么实现？ | [04-suggest-儿童联想搜索.md](./04-suggest-儿童联想搜索.md) |
| REST / gRPC 契约从哪看？ | [../03-接口与集成/README.md](../03-接口与集成/README.md)、[../../api/rest/README.md](../../api/rest/README.md)、[../../api/grpc/README.md](../../api/grpc/README.md) |
| 认证主链专题从哪看？ | [../05-专题分析/01-认证链路：从登录请求到 Token 与 JWKS.md](../05-专题分析/01-认证链路：从登录请求到 Token 与 JWKS.md) |
| 授权主链专题从哪看？ | [../05-专题分析/02-授权判定链路：角色、策略、资源、Assignment、Casbin.md](../05-专题分析/02-授权判定链路：角色、策略、资源、Assignment、Casbin.md) |
| 用户与监护关系当前从哪看？ | [03-user-用户、儿童、Guardianship.md](./03-user-用户、儿童、Guardianship.md)、[../05-专题分析/03-监护关系链路：用户、儿童、Guardianship 的协作.md](../05-专题分析/03-监护关系链路：用户、儿童、Guardianship 的协作.md) |

## 当前业务域地图

| 业务域 | 当前职责 | 新正文位置 |
| ---- | ---- | ---- |
| `authn` | 登录、账户、Token、JWKS、安全基线 | [01-authn-认证、Token、JWKS.md](./01-authn-认证、Token、JWKS.md) |
| `authz` | 角色、策略、资源、Assignment、Casbin 规则落地 | [02-authz-角色、策略、资源、Assignment.md](./02-authz-角色、策略、资源、Assignment.md) |
| `user` | 用户、儿童、监护关系 | [03-user-用户、儿童、Guardianship.md](./03-user-用户、儿童、Guardianship.md) |
| `idp` | 第三方身份提供方能力 | 暂未单列，先看 `api/` 合同和装配点 |
| `suggest` | 儿童联想搜索等补充读侧能力 | [04-suggest-儿童联想搜索.md](./04-suggest-儿童联想搜索.md) |

## 与其他层的分工

| 层 | 负责什么 |
| ---- | ---- |
| `00-概览` | 系统地图、术语、阅读路径、事实来源 |
| `01-运行时` | 进程、gRPC、mTLS、健康检查、装配 |
| `02-业务域` | 各业务域当前能力、边界、主模型、当前风险 |
| `03-接口与集成` | 契约解释层、接入边界、合同导航 |
| `04-基础设施与运维` | 六边形、CQRS、部署、配置、Makefile、迁移 |
| `05-专题分析` | 跨层主链路、重点设计与当前保证 |

## 当前约定

1. 新正文优先写在 `docs/02-业务域/`，不再回写旧 `认证域 / 授权域 / 用户域` 目录。
2. `authn / authz / user` 的旧目录已进入 `_archive/`。
3. 业务域正文默认采用：
   `本文回答 -> 30 秒结论 -> 重点速查 -> 当前实现 -> 当前边界`
4. 对尚未完成的能力，一律写成：
   `已实现 / 待补证据 / 待开发`
