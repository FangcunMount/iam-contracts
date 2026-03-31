// Package tenant 定义 IAM 多租户在请求/Casbin/授权表中的统一租户标识约定。
package tenant

// DefaultID 未在请求上下文显式指定租户时使用的租户 ID（与 tenants.id、Casbin domain、authz.tenant_id 对齐）。
const DefaultID = "fangcun"
