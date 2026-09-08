# IAM v5 切换验证记录

> 状态：已实现 · 生产核心切换已完成。2026-09-08 19:16 进入维护，19:35 恢复流量；模拟数据程序 19:38 完成配套发布。时间为北京时间。

## 生产版本与发布

| 组件 | 生产提交 | 发布证据 |
| --- | --- | --- |
| IAM / SDK v5.0.0 | `cdc69fe0bd5004d535ebfa5cdbb5ee8b3e18a201` | [生产发布](https://github.com/FangcunMount/iam/actions/runs/34220317961)、[v5.0.0](https://github.com/FangcunMount/iam/releases/tag/v5.0.0) |
| QS API / Collection / Worker | `11cb1006f8738ee45137e2029ce2171d2818ff28` | [主分支 CI](https://github.com/FangcunMount/qs-server/actions/runs/34219466331)、[完整发布](https://github.com/FangcunMount/qs-server/actions/runs/34220842699) |
| 管理前端 | `923804585fda5d33937f31ae7691d4c816c77133` | [生产发布](https://github.com/FangcunMount/qs-operating-system/actions/runs/34220327428) |
| seeddata-runner | `dc130c284d86ee28fbed042876f674b4541f062b` | [生产发布](https://github.com/FangcunMount/seeddata-runner/actions/runs/34221541673) |

IAM、QS API、两副本 Collection 和前端镜像版本与目标一致且健康。三个 Worker 使用目标镜像并运行，发布流程的 Worker 治理与观测检查通过。模拟数据程序完成不可变镜像校验及宿主配置迁移。生产入口健康检查与前端均返回 200。

## 数据治理与登录状态切换

- 用户明确批准备份归档后治理 7 条失效历史孤儿和 3 条普通管理角色的资源目录写授权。停写预检逐项比较完整问题集合，与批准范围完全一致。
- 停写后的完整备份：`iam_backup_20260908_191746.sql.gz`，30,099,750 bytes，SHA256 `3c453485d0345eb0e1eebd6f63de6d65570aca0bcc7231ed37a5cacaaa890042`。
- 10 条记录已完整归档；3 条敏感授权先撤销再移出，普通目录读取能力保留。授权迁移至 32，运行时快照校验通过，统一策略版本为 11。
- 新签名密钥已启用，3 把旧活动/Grace 密钥强制退役；公网 JWKS 仅发布新公钥。保留原私钥审计材料，不重新启用旧密钥。
- 独立授权审计归档 SHA256：`c528e5597845b2546e7cf8f605e0783d6d0fbc1a3be6a4da10d55dd82707f4ef`；切换后密钥表归档 SHA256：`c9d325a7a9122adef08ebcd28b9957c615969cb292b6dd890b0b66741266c1bc`。
- IAM 专属状态删除计数：Session 324、RefreshToken 324、身份会话索引 80,784、用户会话索引 80,795、已消费刷新令牌 2。复查所有目标类别剩余为 0；未清空共享 Redis。
- 私密备份及逐步报告位于服务器受限目录 `/opt/backups/iam/tenant-retirement-20260908`，不提交原始主体权限、令牌、用户数据或私钥。

回滚必须整体恢复旧版本、配置与授权备份，并覆盖恢复切换后的密钥状态，维持旧密钥退役和全员重新登录。不得恢复旧 Session 或 RefreshToken；恢复数据库前先导出切换后的授权审计差异。

## 实际验收

- 旧 AccessToken：IAM 在线拒绝；官方 SDK v5 使用公网新 JWKS 本地拒绝。样本在切换前刷新并验证有效，拒绝不是依赖自然过期。
- 旧 RefreshToken：无法刷新。新登录成功，新 JWT 不含 `tenant_id` 或 `tenant_domain`。
- 新令牌：`iam-api`、`qs-api`、`collection-api` 在线验证通过，SDK 本地验证通过。
- 新令牌访问 IAM 自身身份查询和 Collection 档案查询通过；普通消费者访问 QS 管理入口被拒绝。
- 刷新轮换、刷新后验证、注销、注销后在线拒绝均通过；验收登录已注销。
- IAM readiness 中 AuthN、AuthZ、JWKS、MySQL、Redis、Outbox、会话撤销与 Suggest 均正常；策略版本滞后为 0，快照刷新成功。
- 额外只读 QS 授权矩阵在策略版本 11 完成 12 项验证：管理员、评估员、计划管理员、普通员工的来源约束，以及缺失属性、错误类型和强制重试权限均通过。

统一空间带来的明确变化：同时拥有平台管理与 QS 管理角色的主体，其允许决定可命中 `platform_admin` 的授权；QS 应用快照仍仅返回 QS 角色。旧验收器强制要求命中 `qs:admin` 导致误报，已用独立修正工具验证；工具修正由 [QS PR 74](https://github.com/FangcunMount/qs-server/pull/74) 跟踪。该工具不修改线上授权事实或规则。

## 本地与副本验证

- MySQL 8.0.44：空库初始化、升级、错误预检阻断、准备指纹失效、重复迁移及备份恢复演练通过。
- 生产备份副本：归档 10 条记录后，快照版本 11、角色 9、分配 135、Grant 97、继承 8；再次恢复后事实指纹相同。
- IAM、QS、SDK、seeddata-runner 全量测试，前端测试及生产构建通过；IAM 与 QS 主分支 CI 通过。
- IAM 运行时、keyset、Redis、管理保护与 QS 授权缓存 race 检查通过；MySQL 继承环、删除与 Grant、目录更新与 Grant、撤销和 Replace 并发测试通过。

## 切换中处理的问题

- QS 固定日期集成测试被 MongoDB TTL 清理提前删除夹具；改为当前 UTC 后，本地重复测试及 PR/主分支 E2E 通过。
- 生产预检 CLI 的 JSON 后附诊断文本，包装脚本严格解析 JSON 与已知诊断后继续；未跳过完整问题集合比对。
- QS 自动任务沿用触发时暂停变量而跳过部署，重新手动发布完成；前端镜像仓库短暂连接中断，重试后完成。均未绕过测试或授权检查。
