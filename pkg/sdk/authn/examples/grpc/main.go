package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	authnsdk "github.com/FangcunMount/iam-contracts/pkg/sdk/authn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// 全局验证器
var grpcVerifier *authnsdk.Verifier

// 初始化验证器
func initGRPCVerifier() error {
	cfg := authnsdk.Config{
		JWKSURL:         "https://iam.example.com/.well-known/jwks.json",
		AllowedAudience: []string{"grpc-service"},
		AllowedIssuer:   "https://iam.example.com",
	}

	var err error
	grpcVerifier, err = authnsdk.NewVerifier(cfg, nil)
	if err != nil {
		return fmt.Errorf("初始化验证器失败: %w", err)
	}

	log.Println("✅ 验证器初始化成功")
	return nil
}

// AuthInterceptor 一元 RPC 认证拦截器
func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 1. 跳过不需要认证的方法
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// 2. 提取 metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// 3. 提取 authorization
		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		// 4. 提取 Bearer token
		token := tokens[0]
		if !strings.HasPrefix(token, "Bearer ") {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}
		token = strings.TrimPrefix(token, "Bearer ")

		// 2. 验证 token
		resp, err := grpcVerifier.Verify(ctx, token, nil)
		if err != nil {
			log.Printf("Token 验证失败: %v", err)
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		// 6. 将用户信息注入上下文
		ctx = context.WithValue(ctx, "user_id", resp.Claims.UserId)
		ctx = context.WithValue(ctx, "tenant_id", resp.Claims.TenantId)
		ctx = context.WithValue(ctx, "account_id", resp.Claims.AccountId)

		// 7. 调用实际的处理函数
		return handler(ctx, req)
	}
}

// StreamAuthInterceptor 流式 RPC 认证拦截器
func StreamAuthInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// 跳过公开方法
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		// 提取并验证 token
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization token")
		}

		token := strings.TrimPrefix(tokens[0], "Bearer ")
		resp, err := grpcVerifier.Verify(ctx, token, nil)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		// 创建包装的 ServerStream
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          enrichContext(ctx, resp.Claims),
		}

		return handler(srv, wrappedStream)
	}
}

// wrappedServerStream 包装 ServerStream 以注入自定义上下文
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// enrichContext 将 token claims 注入上下文
func enrichContext(ctx context.Context, claims interface{}) context.Context {
	// 这里简化处理，实际项目中可以根据 claims 类型添加更多信息
	return ctx
}

// isPublicMethod 判断是否为公开方法
func isPublicMethod(method string) bool {
	publicMethods := []string{
		"/api.HealthService/Check",
		"/api.PublicService/GetVersion",
	}

	for _, pm := range publicMethods {
		if method == pm {
			return true
		}
	}
	return false
}

// getUserID 从上下文获取用户 ID
func getUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

// getTenantID 从上下文获取租户 ID
func getTenantID(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value("tenant_id").(string)
	return tenantID, ok
}

// ===== 示例 gRPC 服务定义 =====

// 这里只是示例，实际项目中应该使用 protobuf 定义

type ExampleServer struct{}

func (s *ExampleServer) GetUserInfo(ctx context.Context, req interface{}) (interface{}, error) {
	userID, ok := getUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "user id not found")
	}

	tenantID, _ := getTenantID(ctx)

	log.Printf("处理请求 - 用户: %s, 租户: %s", userID, tenantID)

	return map[string]string{
		"user_id":   userID,
		"tenant_id": tenantID,
		"message":   "success",
	}, nil
}

func (s *ExampleServer) PublicMethod(ctx context.Context, req interface{}) (interface{}, error) {
	log.Println("处理公开方法请求")
	return map[string]string{
		"message": "this is a public method",
	}, nil
}

// ===== 高级拦截器示例 =====

// TenantValidationInterceptor 租户验证拦截器
// 必须在认证拦截器之后使用
func TenantValidationInterceptor(allowedTenants []string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		tenantID, ok := getTenantID(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "tenant information missing")
		}

		// 验证租户权限
		allowed := false
		for _, t := range allowedTenants {
			if t == tenantID {
				allowed = true
				break
			}
		}

		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "tenant not allowed")
		}

		return handler(ctx, req)
	}
}

// LoggingInterceptor 日志拦截器
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		userID, _ := getUserID(ctx)
		log.Printf("gRPC 调用: %s, 用户: %s", info.FullMethod, userID)

		resp, err := handler(ctx, req)

		if err != nil {
			log.Printf("gRPC 调用失败: %s, 错误: %v", info.FullMethod, err)
		} else {
			log.Printf("gRPC 调用成功: %s", info.FullMethod)
		}

		return resp, err
	}
}

func main() {
	// 1. 初始化验证器
	if err := initGRPCVerifier(); err != nil {
		log.Fatal(err)
	}

	// 2. 创建 gRPC 服务器
	server := grpc.NewServer(
		// 使用拦截器链
		grpc.ChainUnaryInterceptor(
			AuthInterceptor(),
			TenantValidationInterceptor([]string{"tenant-123", "tenant-456"}),
			LoggingInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			StreamAuthInterceptor(),
		),
	)

	// 3. 注册服务（示例）
	// pb.RegisterYourServiceServer(server, &ExampleServer{})

	// 4. 启动服务器
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("监听失败: %v", err)
	}

	log.Println("🚀 gRPC 服务器启动在 :50051")
	log.Println("拦截器链: Auth → TenantValidation → Logging")

	if err := server.Serve(listener); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
