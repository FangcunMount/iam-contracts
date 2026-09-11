# AuthZ SDK

通过 `client.Authz().Allow(ctx, subject, resource, action)` 执行资源与动作检查；需要命中授权及策略版本时使用普通 `Check`。

```go
allowed, err := client.Authz().Allow(ctx, "user:42", "qs:evaluation:collection:assessments", "retry")
if err != nil { return err }
if !allowed { return errors.New("permission denied") }
```

主体必须来自可信身份上下文。SDK 复用现有 mTLS 和错误包装；业务服务仍需校验机构、受试者关系和对象状态，列表应在分页与统计前执行业务范围过滤。

`CheckObject` 已弃用，保留源码符号但直接返回“条件授权已退役”错误，不发起 RPC，也不转换为无条件允许。gRPC 非空 object_context 返回 InvalidArgument。协议字段号暂留，后续主版本再删除公共符号。

快照只产生 UNCONDITIONAL 权限。消费者收到旧 OBJECT_CHECK_REQUIRED 模式时不得将其当作动作权限，应刷新数据并保持拒绝。多角色授权仍取并集。

参见 [AuthZ 维护手册](../../../docs/02-业务模块/03-AuthZ/08-条件授权退役维护手册.md)。

快照兼容字段 `roles` 与 `direct_roles` 使用相同应用范围过滤并返回直接角色集合；角色继承已退役。

`ReplaceManagedAssignments` 只替换调用服务获准管理的角色集合，保留集合之外的分配；方法 ACL 与服务角色白名单共同生效。授权快照的 `direct_roles` 表示直接分配，不包含继承。
