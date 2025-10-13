# IAM API 指南

本目录包含 IAM 的两类接口协议：

- **REST**（/api/rest）：面向运营后台、前端、脚本调用，资源导向、便于调试与审计；
- **gRPC**（/api/grpc）：面向服务间高 QPS/低时延调用，主要用于**鉴权判定**与**监护读侧**。

## 什么时候用哪一个？

| 能力 | 建议协议 |
|---|---|
| PDP 判权（Allow/AllowOnActor/批量） | gRPC |
| 监护判定（IsGuardian / ListChildren） | gRPC |
| User/Child/Guardianship 的创建、更新、查询、注册 | REST |
| “我的孩子” | REST（BFF 可转调 gRPC） |

## 安全与通用约定

- **认证**：JWT（`Authorization: Bearer <token>`）；gRPC 放在 `authorization` metadata。
- **传输**：REST 走 HTTPS；gRPC 走 mTLS。
- **幂等**：REST 的 `POST` 支持 `X-Idempotency-Key`；gRPC 幂等由调用方重试 + 语义保证。
- **追踪**：`X-Request-Id`（REST）；`x-request-id`（gRPC metadata）。

## /api 文档结构

```text
api/
├─ README.md                # 顶层说明（何时用 REST / gRPC）
├─ rest/
│  └─ identity.v1.yaml      # OpenAPI 3.1（User/Child/Guardianship）
└─ grpc/
   ├─ iam.authz.v1.proto    # PDP 判权（Allow/AllowOnActor/Batch/Explain）
   └─ iam.identity.v1.proto # Identity 读侧（GetUser/GetChild/IsGuardian/ListChildren）
```
