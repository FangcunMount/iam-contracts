// Package mtls 提供 mTLS 双向认证配置和管理功能
//
// 此包提供通用的 mTLS 配置和证书管理能力，适用于任何 gRPC 服务。
// 设计为可提取到 component-base 的独立模块。
//
// 功能特性:
//   - 服务端和客户端 TLS 配置构建
//   - 证书自动重载和热更新
//   - 客户端证书验证（CN/OU/SAN 白名单）
//   - 服务身份提取
//
// 使用示例:
//
//	cfg := mtls.DefaultConfig()
//	cfg.CertFile = "/path/to/server.crt"
//	cfg.KeyFile = "/path/to/server.key"
//	cfg.CAFile = "/path/to/ca.crt"
//
//	creds, err := mtls.NewServerCredentials(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	server := grpc.NewServer(creds.GRPCServerOption())
package mtls
