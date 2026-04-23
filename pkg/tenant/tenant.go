// Package tenant 定义 IAM 多租户在请求/Casbin/授权表中的统一租户标识约定。
package tenant

// DefaultID 未在请求上下文显式指定租户时使用的租户 ID（与 tenants.id、Casbin domain、authz.tenant_id 对齐）。
const DefaultID = "fangcun"

// PlatformID 是平台控制面的固定租户 / domain 标识。
const PlatformID = "platform"

// DefaultTenantID 是当前默认业务租户在 JWT / org 语义上的数值 ID。
// 这与 system bootstrap SQL 里 fangcun 租户当前的 org_id=1 约定对齐。
const DefaultTenantID uint64 = 1
