# 服务间认证：mTLS 与 ACL

IAM v4 使用 mTLS 建立服务身份，ACL 决定服务可调用的 RPC，业务 AuthZ 决定资源访问权限。

客户端通过 `sdk.Config.TLS` 配置受信任 CA、客户端证书和私钥，复用 `sdk.Client` 连接，直接传递请求 context。无需申请服务 JWT，也没有续期 helper 或服务 Authorization metadata。

```go
client, err := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "iam.example.com:9090",
    TLS: &sdk.TLSConfig{
        Enabled: true,
        CACert: "/etc/iam/ca.pem",
        ClientCert: "/etc/iam/client.pem",
        ClientKey: "/etc/iam/client-key.pem",
    },
})
if err != nil { return err }
defer client.Close()
```

证书必须在有效期内且受服务端信任，证书服务身份必须匹配 ACL。证书本地读取仅用于诊断；真实 RPC 的服务端校验才是认证证据。轮换证书时遵循部署证书重载机制。

用户 AccessToken 验证、业务授权和 delegated subject proof 仍按各自契约执行。服务身份不能代替被委托用户身份。

## v3 迁移记录

ServiceToken、IssueServiceToken 和 ServiceAuthHelper 已从 v4 删除。旧 token 不能作为 v4 的用户凭证使用；旧客户端不能调用已删除 RPC。IAM 和依赖该能力的服务必须统一升级或整体回滚，不能混合运行。
