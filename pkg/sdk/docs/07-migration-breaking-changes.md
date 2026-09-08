# IAM Go SDK v5 迁移说明

本次升级要求 IAM 服务端、SDK、QS 和管理前端配套切换，所有用户重新登录。旧版本迁移记录保存在仓库的历史文档中。

## 接口版本

| 接口 | 新版本 |
| --- | --- |
| Go module | `github.com/FangcunMount/iam/v5`，`v5.0.0` |
| AuthN REST | `/api/v3/authn` |
| AuthN gRPC | `iam.authn.v3` |
| AuthZ REST | `/api/v4/authz` |
| AuthZ gRPC | `iam.authz.v4` |
| Identity、IDP | 保持 v2 |
| JWKS | 保持 `/api/v2/.well-known/jwks.json` |

AuthN REST 登录客户端位于 `pkg/sdk/auth/loginv3`。其他公开入口包括 `pkg/sdk`、`config`、`auth/client`、`auth/jwks`、`auth/verifier`、`authz`、`identity`、`idp` 和 `errors`。旧主版本的入口不提供别名。

## 登录、声明与验证

身份核验、登录准入、创建会话和颁发令牌是四个独立环节。登录请求和令牌声明不再包含授权分区。用户名使用 default Realm；外部身份提供方 Realm 和业务 OrgID 保持原义。

本地和远程验证必须配置非空 ExpectedAudience。IAM 使用 iam-api，QS 和 collection 分别使用 qs-api、collection-api。多个期望受众保持任意一个匹配语义；空列表、空元素均为参数错误。不从令牌内容推导期望受众。

业务组织读取 `TokenClaims.OrgID` 或 `BusinessOrgID()`。未知扩展字段不会被解释为组织或授权事实。角色和权限不放入 JWT；请使用当前 AuthZ 策略检查。

切换时旧签名密钥退役，旧 Session 与 RefreshToken 清理，因此本地验签、远程验证和刷新都不能继续使用旧登录态。

## 授权与角色分配

Check、Snapshot、GrantAssignment、RevokeAssignment、ReplaceManagedAssignments 移除 domain 参数。Subject、Resource、Action、AppName 与受信对象上下文仍保留。资源标识中的业务模块段不是授权分区，不应删除。

角色名称全局唯一，公开查询使用名称或 ID，内部关系使用 RoleID。原 platform/super_admin 改为 platform_admin；原普通 super_admin 保留名称；tenant_admin 改为 iam_admin。不要使用这些名称绕过权限校验。

Role DTO 返回 management_protection：standard 或 protected。受保护角色及其关联事实的管理需要原始操作权限与 roles/manage_protected。普通角色不能继承受保护角色，也不能获得覆盖敏感能力的通配授权。

QS 批量替换只修改部署配置中的可管理角色集合，集合外分配保持不变。服务身份来自 mTLS，不能通过 ChangedBy 或请求参数声明 admin。

## 全量档案与缓存

全量列表能力为 profiles/list_all；全量手机号检索为 profiles/search_by_mobile_all。普通范围继续保留 OrgID、操作人、关联档案及 search_by_mobile 的检查。

授权快照缓存键为 user + app；策略事件使用 iam.authz.version_changed.v2，主题 iam.authz.version.v2，载荷仅含全局版本。事件到达后，低版本快照失效；并发请求迟到的低版本结果同样拒绝。

## 切换与回滚

先备份、预检、迁移并核对权限差异，再启用新密钥、退役旧密钥、清理登录状态和全部相关缓存，最后更新全部实例及调用方。具体操作见仓库 AuthZ 的安全加固与发布验收文档。

不提供恢复默认授权分区或关闭受众校验的兼容开关。回滚需配套恢复版本、配置和授权数据库备份，保留已退役密钥状态，并继续要求重新登录。
