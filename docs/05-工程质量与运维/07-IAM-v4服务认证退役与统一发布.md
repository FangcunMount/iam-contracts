# IAM v4 服务认证退役与统一发布

> 状态：已实现 · 本地代码与契约测试已完成；版本发布和线上统一切换尚未开始。

## 当前交付状态

本地代码删除 ServiceToken 服务认证方式，Go module 改为 `github.com/FangcunMount/iam/v4`。本记录不是发布成功证明：尚未提交或推送本批改动、创建 v4.0.0 标签、构建发布镜像或切换线上流量。

服务调用责任链为 **mTLS → 服务身份 → ACL → 业务授权**。AccessToken、RefreshToken、Session、JWKS、撤销存储和用户委托证明继续按原合同工作。不安排数据库迁移，不清空共享 Redis 或密钥。

## 兼容性与消费者

| 消费者 | 处理 | 当前证据 |
| --- | --- | --- |
| IAM 服务端与 SDK | 删除签发、验证支持、helper、配置与 RPC；发布目标 v4.0.0 | 本地实现和测试 |
| QS apiserver、管理工具 | IAM v4、mTLS、无服务 Bearer；保留授权启动探测 | 本地实现和真实握手测试 |
| collection-server | IAM v4、mTLS；保留 delegated subject signer，增加 ProfileLink 启动探测 | 本地实现和真实握手测试 |
| qs-worker | QS 仓库中相关 IAM 导入随 module 迁移，原 mTLS 调用保持 | 全仓构建与测试 |
| seeddata-runner | 扫描发现 v3 用户登录 SDK；没有发现服务令牌调用 | 静态调用与接口检查，未验证线上行为 |
| 其他部署、覆盖配置、外部消费者 | 必须逐项核实 | 尚未获得线上清单，不得按零消费者处理 |

protobuf 保留删除枚举的名称和编号 3；其他 RPC 包名和路径保持不变。旧服务 JWT 和未知类型无法显式重新启用，不能默认映射成用户 AccessToken。旧签发 RPC 不存在。历史 v3 用户 SDK 不因 Go module 路径变化自动失去未变更的线协议兼容性。

## 本地验收与复现

已通过两仓库全仓单元测试（IAM 160 个测试包、QS 352 个测试包）、关键认证链路 race 检查、真实 mTLS 契约和生产运行时就绪方法测试。两仓库完整构建、integration-tag 编译也已通过；integration-tag 编译不等于真实数据库集成测试已执行。文档检查与退役检查已通过。

真实握手测试覆盖合法证书无 Bearer、伪造 Bearer 不改变身份、未知身份 ACL 拒绝、缺失/过期/不受信任证书拒绝。测试使用临时证书和本地端口；业务存储协作者使用测试替身，不代表线上数据或全业务端到端验收。

IAM 本地命令：

```sh
go test ./...
go build ./...
go test -tags=integration -run '^$' ./...
go test -race ./pkg/sdk/auth/verifier ./internal/apiserver/domain/authn/token ./internal/apiserver/infra/token/jwt ./internal/apiserver/transport/grpc/service/authn ./internal/apiserver/transport/grpc/service/authz
python3 scripts/check_service_token_retirement.py
python3 scripts/check-docs-facts.py
bash scripts/proto/generate.sh
```

协议生成需保持无额外差异；生成插件版本已固定。模型图 SVG/PNG 已同步更新。

v4.0.0 尚未发布时，在 QS 仓库执行 `bash scripts/verify-iam-v4.sh ../iam`。脚本创建临时 Go workspace 和仅本地有效的替换，不往 go.mod 提交开发机路径。发布标签后运行普通依赖下载，补齐 v4.0.0 的校验和，再运行独立 CI。

QS 文档中受影响的六条旧提交验收锚点标记为 needs_review，并保留历史证据；提交本次迁移后重新运行检查、绑定真实 source SHA，不能借用旧 SHA 宣称 v4 已验收。

## 发布前置条件与版本清单

发布负责人须在切换前填写并核验：

- IAM 实际提交 SHA、v4.0.0 标签目标与不可变镜像 digest。
- QS 实际提交 SHA，以及 apiserver、collection-server、worker、管理工具的不可变构建标识；其 SDK 依赖必须为已发布 v4。
- 上一整套服务镜像 digest、配置备份和证书部署路径；保留已验证的整体回滚产物。
- 所有部署实例、定时任务、运维脚本与外部调用方清单，确认没有遗留签发请求或服务 Bearer 注入。
- 每个环境实际生效的 mTLS、强制客户端证书、默认拒绝 ACL、允许身份和业务 RPC；不能仅检查仓库 YAML。
- 预发布环境的冷缓存快照、对象权限、角色写入及回读、用户登录/刷新/撤销和委托证明正负向测试。

当前上述线上版本清单和预发布证据尚未填写，发布状态为未开始。

## 统一切换与回滚

1. 发布并验证 IAM v4 SDK，完成全部服务的不可变构建；SDK 标签发布与线上服务切换分别记录。
2. 在约定窗口暂停入口流量、后台调用和相关管理任务，等待在途请求结束并停止旧消费者。
3. 部署新 IAM、QS apiserver、collection-server 以及本批涉及的 worker/工具和配套配置；不混用旧消费者与新 IAM。
4. 检查全部就绪探测，执行真实证书与业务冒烟，确认成功和拒绝语义都符合预期，然后恢复流量。
5. 观察 TLS 失败、ACL 拒绝、旧 RPC 的 Unimplemented、启动失败、授权快照/对象鉴权/角色写入错误。出现旧签发请求即重新检查遗漏消费者。
6. 任一必需验收失败时保持或重新暂停流量，整体恢复上套二进制和配置，验证后恢复。禁止单独回滚依赖旧签发器的 QS 或 collection-server。

新版本立即拒绝旧服务令牌，无需等待其自然过期。整体回滚旧版本会恢复旧认证行为，须在事故记录中明确说明；新版本不保留兼容开关或签发分支。
