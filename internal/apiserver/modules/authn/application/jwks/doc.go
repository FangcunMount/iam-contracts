// Package jwks 提供 JWKS（JSON Web Key Set）应用层服务
//
// 应用层职责：
//  1. 事务边界管理：协调多个聚合根的操作
//  2. 依赖注入：组装领域服务和基础设施适配器
//  3. 业务流程编排：实现跨聚合的复杂业务流程
//  4. 日志和监控：记录应用级别的操作日志
//  5. 错误转换：将领域错误转换为应用错误
//
// 应用服务 vs 领域服务：
//   - 领域服务：纯业务逻辑，无副作用（如日志、事务）
//   - 应用服务：包含技术关注点（事务、日志、监控）
//
// 架构分层：
//
//	Interface Layer (RESTful/gRPC)
//	     ↓
//	Application Layer (本包)
//	     ↓
//	Domain Layer (领域服务 + 端口)
//	     ↓
//	Infrastructure Layer (适配器实现)
package jwks
