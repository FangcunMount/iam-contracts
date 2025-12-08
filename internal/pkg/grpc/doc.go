// Package grpc 提供通用的 gRPC 服务器基础设施
//
// 本包为 apiserver 提供 gRPC 服务器的通用能力，包括：
//   - Server: gRPC 服务器封装，支持 TLS、健康检查、反射服务
//   - Config: 服务器配置，包括绑定地址、消息大小限制、连接管理等
//   - 拦截器: 日志、恢复、请求 ID 等通用拦截器
//
// 使用示例：
//
//	config := grpc.NewConfig()
//	config.BindPort = 9090
//	config.EnableHealthCheck = true
//
//	server, err := config.Complete().New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 注册服务
//	pb.RegisterMyServiceServer(server.Server, myService)
//
//	// 启动服务器
//	server.Run()
//
// 对于 mTLS、ACL 等高级安全功能，可使用 pkg/grpc/mtls 和
// pkg/grpc/interceptors 中的组件来扩展。
package grpc
