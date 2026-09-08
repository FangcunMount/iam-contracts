# AuthN：身份核验与登录态管理

> 状态：已实现 · 认证模型、核心用例、Session/Token/JWKS 与已知失败窗口已按当前实现复核。

AuthN 证明“当前请求者是谁”，维持认证会话，并把认证结果转化为可验证令牌。它不维护 User/Profile 档案，不配置外部身份源，也不回答资源授权问题。

从领域职责看，AuthN 包含三组能力；这组分类与下述登录流程的四个环节不同：

```text
认证关系管理 / 身份核验与登录准入 / 登录态管理
```

- 建立认证关系：用 `LoginIdentity`、`Credential` 表达“系统凭什么认识你”；
- 身份核验与登录准入：用 `Challenge`、`Authenticator`、`AuthDecision`、`Principal`、`AdmissionPolicy` 表达“如何确认现在是你”；
- 登录态管理：用 `登录结果`、`Session`、Token 与 `SignIn`、`Verifier`、`Refresher`、`Revoker` 表达“如何让系统持续相信是你”。

JWT、JWKS 和 Redis 是登录态管理相关的适配实现，不是独立的登录环节。

AuthN 是身份核验及登录态管理模块。其登录用例统一称为：

**身份核验 → 登录准入 → 会话建立 → 令牌颁发。**

| 环节 | 职责 | 输出 |
| --- | --- | --- |
| 身份核验 | 核验请求者是否控制某个登录身份，并确认对应主体 | AuthDecision；通过时包含 Principal |
| 登录准入 | 判断主体当前是否允许建立或维持登录态 | admission.Decision |
| 会话建立 | 创建并保存可管理的登录状态 | Session |
| 令牌颁发 | 交付访问与续期凭证，并保存初始刷新令牌 | TokenPair |

“身份核验成功”只表示第一环节成功；“登录成功”表示四个环节全部完成。Authenticator、Authenticate、AuthDecision 等英文代码标识保留，AuthN 模块名称保持不变。

## 按任务阅读

| 你要回答的问题 | 首选入口 |
| --- | --- |
| 第一次理解 AuthN | [模块总览](00-模块总览.md) → [局部领域模型](01-领域模型与认证策略.md) |
| 修改密码/OTP/provider 登录 | [Login 主链路](04-关键链路-Login登录认证.md) |
| 修改绑定、解绑或最近认证 | [Linking 主链路](03-关键链路-Linking登录身份绑定.md) |
| 修改签发、刷新、撤销 | [Token 主链路](05-关键链路-Token签发刷新吊销.md) |
| 理解原始认证时间、Session 寿命和旧格式 | [Session/Token 模型](03-Session-Token与JWKS.md) |
| 排查 key rotation、JWKS 和备份 | [签名密钥生命周期与 JWKS 发布](06-关键链路-JWKS与本地验签.md) |

## 主题归属与维护规则

每项规则在下表指定的文档讲透，其他文档只保留必要摘要并回链。短文用于建立模型，链路文档负责具体执行；修改实现时同时核对主文、摘要和图，不能只补一段文字。

| 主题 | canonical 文档 |
| --- | --- |
| 领域对象、生命周期与职责 | 01 领域模型 |
| SignUp Prepare/UoW、创建与修复 | 02 注册登录与身份绑定 |
| Link/Unlink 最近认证、证明、幂等、会话影响 | 03 Linking 链路 |
| SignIn/策略/凭据记录与错误 | 04 Login 链路 |
| Grant/Verify/Refresh/Revoke 执行与补偿 | 05 Token 链路 |
| 上下文投影、寿命、历史兼容退役门禁 | 03 Session、Token 与 JWKS |
| 签名密钥状态、发布、管理与运行 | 06 JWKS 链路 |
| 跨模块事实交换、分层与改动范围 | 07 模块边界、08 代码索引 |

文件保留既有编号和链接，其中两个 03 分别承担模型与绑定链路；按标题和上述主题定位。扩展设计必须明确标注尚未实现，通用建议不得放进当前执行时序。图须说明是对象关系、成功路径、含失败分支的执行时序，还是业务生命周期。

## 完整阅读路径

1. [模块总览](00-模块总览.md)：先理解认证关系、身份核验与登录准入、登录态管理的职责划分。
2. [领域模型与身份核验策略](01-领域模型与认证策略.md)：区分 LoginIdentity、Credential、Challenge、Principal、Session、登录结果 和 Token 概念族。
3. [注册、登录与身份绑定](02-注册登录与身份绑定.md)：理解三条写链路的事务、幂等和并发边界。
4. [Session、Token 与 JWKS](03-Session-Token与JWKS.md)：理解在线状态、刷新轮换、撤销与两种验签语义。
5. [登录身份绑定](03-关键链路-Linking登录身份绑定.md)：深入理解绑定、解绑和最后一个 active identity 的并发保护。
6. [登录认证](04-关键链路-Login登录认证.md)：深入理解身份核验策略、失败记录和锁定语义。
7. [Token 签发、刷新与吊销](05-关键链路-Token签发刷新吊销.md)：深入理解 Redis 状态机和失败窗口。
8. [JWKS 与本地验签](06-关键链路-JWKS与本地验签.md)：深入理解密钥生命周期和离线验证边界。
9. [模块边界](07-模块边界-AuthN与Identity-IDP-AuthZ.md)：理解 User、外部身份和 Subject 的跨模块转换。
10. [分层架构与代码索引](08-分层架构与代码索引.md)：定位新增 provider、claims、refresh、JWKS 的完整修改面。

跨模块内容：

- [Identity：User、Profile 与状态一致性](../01-Identity/README.md)
- [IDP：外部身份源](../04-IDP/README.md)
- [AuthZ：资源授权](../03-AuthZ/README.md)
- [密码学、密钥与令牌](../../03-基础设施/04-密码学密钥与令牌.md)
- [Redis 与缓存一致性](../../03-基础设施/02-Redis与缓存一致性.md)

## 责任边界

```text
IDP 解析外部 provider 身份
  -> AuthN 映射 LoginIdentity，验证证明并形成 Principal
  -> Admission 读取 Identity User / AuthN LoginIdentity 当前状态
  -> SignIn 颁发 登录结果
  -> AuthZ 对 Principal 对应主体做资源授权
```

## 当前实现要特别记住的七点

- SignUp 的外部身份解析在事务外，本地 User/LoginIdentity/Credential 在一个 MySQL UoW 中提交。
- SignIn 在应用层分别调用 AdmissionPolicy、SessionCreator 和 InitialTokenIssuer，并负责新会话的失败补偿；续期和登出仍分别由 Refresher、Revoker 处理。
  准入返回 Decision；只有明确通过才创建 Session，随后颁发并保存初始令牌。
- 用户 access token 是 RS256 JWT，但 IAM 在线验证仍检查撤销标记、Session 和主体状态。
- Refresh token 使用 Redis Lua 原子轮换；当前先延长 Session，再轮换 token，失败时存在 TTL 已变化的窗口。
- SDK 本地 JWKS 验签不具备 IAM 在线验证的即时撤销语义。
- AuthN 管理路由不按管理员角色名旁路：JWKS 与 Session 操作使用明确的 AuthZ Resource/Action，并依次检查当前授权域 与平台域。

## 代码入口

- domain：`internal/apiserver/domain/authn`
- application：`internal/apiserver/application/authn`
- MySQL infra：`internal/apiserver/infra/mysql/loginidentity`、`credential`、`uow/authn`
- Redis infra：`internal/apiserver/infra/cache/redis`
- transport/container：`internal/apiserver/transport/{rest,grpc}`、`internal/apiserver/container/authn`
- client SDK：`pkg/sdk/auth`

## 验证

```bash
go test ./internal/apiserver/domain/authn/... ./internal/apiserver/application/authn/... ./pkg/sdk/auth/...
```
