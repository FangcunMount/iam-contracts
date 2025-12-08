// Package interceptors 提供通用的 gRPC 拦截器
//
// 此包提供可复用的 gRPC 拦截器，适用于任何 gRPC 服务。
// 设计为可提取到 component-base 的独立模块。
//
// 包含的拦截器:
//   - mTLS 认证拦截器
//   - 凭证验证拦截器 (Bearer/HMAC/API Key)
//   - 审计日志拦截器
//   - 监控指标拦截器
//
// 使用示例:
//
//	server := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(
//	        interceptors.MTLSInterceptor(),
//	        interceptors.CredentialInterceptor(extractor, validator),
//	        interceptors.AuditInterceptor(nil),
//	    ),
//	)
package interceptors
