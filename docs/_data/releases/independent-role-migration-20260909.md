# 独立业务角色迁移执行记录

> 状态：已实现 · 最终清理代码已落地；生产 apply 与删表仍按发布步骤执行。

## 数据依据

2026-09-09 通过 IAM 维护入口只读查询生产 IAM 与 QS。预演覆盖 129 位用户，原直接分配中 staff 124、evaluator 2；迁移目标 operator 125、reviewer 118（按用户合并，不是新增分配数量）。原始逐人报告保存在执行主机受限目录，不将人员身份列表提交仓库。预演没有发现身份映射异常；真正切换前必须冻结写入并重新预演，不能复用本文人数作为执行依据。

## 兼容批次

- IAM `iam-maintenance role-model-migrate preflight|apply|verify|rollback`，使用 MYSQL_* 与 QS_MYSQL_* 环境配置。连接信息不得放在命令参数或报告中。
- schema 33 仅增加迁移清单表；不会自动变更人员权限。
- Apply 必须同时传入 `--writes-stopped` 和经过审阅的 `--fingerprint`；事务内重新读取数据、生成并核对计划，记录前后完整授权事实，递增 PolicyVersion 并写 Outbox。
- QS 兼容开关 `QS_AUTHZ_ROLE_MODEL=independent-v1` 启用后台专业查询的岗位权限检查。空值/legacy 只用于兼容发布期；未知值拒绝访问。
- 运营端对应构建开关 `REACT_APP_AUTHZ_ROLE_MODEL=independent-v1`。进度查询独立返回 DTO。临床身份探测不依赖 staff 角色。

## 最终清理批次（本分支）

- 运行时仅解析直接 Assignment；`roles` 与 `direct_roles` 同值。
- 删除继承领域、仓储、图遍历与管理 handler；旧 REST 路径返回 `410 Gone`。
- `archive-inheritance` + migration `000034` 归档并删除 `authz_role_inheritances`。
- 新库完整链经 fresh stage 直接得到独立角色目标态。

## 切换顺序与验收

1. 发布兼容 IAM、QS 和运营端，保留旧行为开关。核查全部配置消费者以及继承 CRUD 使用情况。
2. 暂停后台业务、授权写入及 QS 人员身份变更。重新 preflight、审阅逐用户权限差异及身份依据，核对指纹后 apply。
3. 切换 QS 与运营端开关；清除旧授权缓存，更新 QS 人员角色投影；确认运行时策略版本收敛和 Outbox 无积压。
4. 验证新岗位的正反权限、条件 retry、医生关系范围、参与者自服务和结果查询缓存；通过后恢复后台服务。
5. 完成回滚演练与调用方核对，再发布不依赖继承表的服务，执行 `archive-inheritance` 后用 `000034` 删除表。

Rollback 只能补偿未被后续授权写入修改的完整事实；冲突时停止。PolicyVersion 必须继续递增。应用查询边界与角色发放成组回滚。删表后须先恢复结构及归档数据，再恢复旧二进制。

## 已验证与未完成

IAM 最终清理相关单测已通过；迁移测试在 SQLite 和本机真实 MySQL 路径具备。运营端与 QS 兼容开关侧此前已验证。

生产版本部署、数据 apply、运行时收敛与真实业务验收尚未完成。
