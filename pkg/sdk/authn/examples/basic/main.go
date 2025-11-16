package main // 基础使用示例

import (
	"context"
	"fmt"
	"log"
	"time"

	authnsdk "github.com/FangcunMount/iam-contracts/pkg/sdk/authn"
)

// 示例 1: 最简单的本地验证
func example1_BasicLocalVerification() {
	fmt.Println("\n=== 示例 1: 基础本地验证 ===")

	// 配置 SDK，只需要 JWKS URL
	cfg := authnsdk.Config{
		JWKSURL: "https://iam.example.com/.well-known/jwks.json",
	}

	// 创建验证器（不传 client 则仅本地验证）
	verifier, err := authnsdk.NewVerifier(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 验证 JWT token
	ctx := context.Background()
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS0yMDI0LTAxIn0..."

	resp, err := verifier.Verify(ctx, token, nil)
	if err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	// 打印验证结果
	fmt.Printf("✅ 验证成功\n")
	fmt.Printf("用户 ID: %s\n", resp.Claims.UserId)
	fmt.Printf("租户 ID: %s\n", resp.Claims.TenantId)
	fmt.Printf("过期时间: %v\n", resp.Claims.ExpiresAt.AsTime())
}

// 示例 2: 带 Audience 和 Issuer 验证
func example2_WithAudienceIssuer() {
	fmt.Println("\n=== 示例 2: 带 Audience 和 Issuer 验证 ===")

	cfg := authnsdk.Config{
		JWKSURL: "https://iam.example.com/.well-known/jwks.json",

		// 配置允许的 audience 列表
		AllowedAudience: []string{"my-app", "admin-panel"},

		// 配置允许的 issuer
		AllowedIssuer: "https://iam.example.com",
	}

	verifier, err := authnsdk.NewVerifier(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

	resp, err := verifier.Verify(ctx, token, nil)
	if err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 验证成功\n")
	fmt.Printf("Issuer: %s\n", resp.Claims.Issuer)
	fmt.Printf("Audience: %v\n", resp.Claims.Audience)
	fmt.Printf("Subject: %s\n", resp.Claims.Subject)
}

// 示例 3: 自定义 JWKS 缓存配置
func example3_CustomJWKSConfig() {
	fmt.Println("\n=== 示例 3: 自定义 JWKS 缓存配置 ===")

	cfg := authnsdk.Config{
		JWKSURL: "https://iam.example.com/.well-known/jwks.json",

		// 自定义 JWKS 刷新间隔
		JWKSRefreshInterval: 3 * time.Minute, // 每 3 分钟刷新一次

		// 自定义 HTTP 请求超时
		JWKSRequestTimeout: 5 * time.Second, // 5 秒超时

		// 自定义缓存 TTL
		JWKSCacheTTL: 15 * time.Minute, // 缓存 15 分钟

		// 自定义时钟偏差容忍度
		ClockSkew: 2 * time.Minute, // 容忍 2 分钟的时间差
	}

	verifier, err := authnsdk.NewVerifier(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

	resp, err := verifier.Verify(ctx, token, nil)
	if err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 验证成功，使用自定义缓存配置\n")
	fmt.Printf("签发时间: %v\n", resp.Claims.IssuedAt.AsTime())
	fmt.Printf("过期时间: %v\n", resp.Claims.ExpiresAt.AsTime())
}

// 示例 4: 本地 + 远程验证
func example4_RemoteVerification() {
	fmt.Println("\n=== 示例 4: 本地 + 远程验证 ===")

	ctx := context.Background()

	// 1. 创建 gRPC 客户端
	client, err := authnsdk.NewClient(ctx, "iam.example.com:8081")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 2. 配置 SDK
	cfg := authnsdk.Config{
		GRPCEndpoint:    "iam.example.com:8081",
		JWKSURL:         "https://iam.example.com/.well-known/jwks.json",
		AllowedAudience: []string{"my-app"},
		AllowedIssuer:   "https://iam.example.com",
	}

	// 3. 创建验证器
	verifier, err := authnsdk.NewVerifier(cfg, client)
	if err != nil {
		log.Fatal(err)
	}

	// 4. 验证 JWT（自动决定是否远程验证）
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

	resp, err := verifier.Verify(ctx, token, nil)
	if err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 验证成功（本地验证）\n")
	fmt.Printf("用户 ID: %s\n", resp.Claims.UserId)
	fmt.Printf("账号 ID: %s\n", resp.Claims.AccountId)
}

// 示例 5: 强制远程验证
func example5_ForceRemoteVerification() {
	fmt.Println("\n=== 示例 5: 强制远程验证 ===")

	ctx := context.Background()

	// 创建客户端和验证器
	client, err := authnsdk.NewClient(ctx, "iam.example.com:8081")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	cfg := authnsdk.Config{
		GRPCEndpoint: "iam.example.com:8081",
		JWKSURL:      "https://iam.example.com/.well-known/jwks.json",
	}

	verifier, err := authnsdk.NewVerifier(cfg, client)
	if err != nil {
		log.Fatal(err)
	}

	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

	// 方式 1: 单次调用强制远程验证
	resp, err := verifier.Verify(ctx, token, &authnsdk.VerifyOptions{
		ForceRemote: true, // 强制使用远程验证
	})
	if err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 验证成功（远程验证）\n")
	fmt.Printf("Token 状态: %v\n", resp.Status)
	fmt.Printf("用户 ID: %s\n", resp.Claims.UserId)
}

// 示例 6: 访问自定义属性
func example6_CustomAttributes() {
	fmt.Println("\n=== 示例 6: 访问自定义属性 ===")

	cfg := authnsdk.Config{
		JWKSURL: "https://iam.example.com/.well-known/jwks.json",
	}

	verifier, err := authnsdk.NewVerifier(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

	resp, err := verifier.Verify(ctx, token, nil)
	if err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 验证成功\n")
	fmt.Printf("标准声明:\n")
	fmt.Printf("  - Token ID: %s\n", resp.Claims.TokenId)
	fmt.Printf("  - Subject: %s\n", resp.Claims.Subject)
	fmt.Printf("  - User ID: %s\n", resp.Claims.UserId)
	fmt.Printf("  - Tenant ID: %s\n", resp.Claims.TenantId)

	// 访问自定义属性
	if len(resp.Claims.Attributes) > 0 {
		fmt.Printf("\n自定义属性:\n")
		for key, value := range resp.Claims.Attributes {
			fmt.Printf("  - %s: %s\n", key, value)
		}
	}
}

// 示例 7: 错误处理
func example7_ErrorHandling() {
	fmt.Println("\n=== 示例 7: 错误处理 ===")

	cfg := authnsdk.Config{
		JWKSURL:         "https://iam.example.com/.well-known/jwks.json",
		AllowedAudience: []string{"my-app"},
		AllowedIssuer:   "https://iam.example.com",
	}

	verifier, err := authnsdk.NewVerifier(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 测试不同的错误场景
	testCases := []struct {
		name  string
		token string
	}{
		{"过期的 Token", "expired_token_here"},
		{"无效的签名", "invalid_signature_token_here"},
		{"错误的 Audience", "wrong_audience_token_here"},
	}

	for _, tc := range testCases {
		fmt.Printf("\n测试: %s\n", tc.name)
		_, err := verifier.Verify(ctx, tc.token, nil)
		if err != nil {
			handleVerifyError(err)
		} else {
			fmt.Println("✅ 验证成功")
		}
	}
}

// handleVerifyError 处理验证错误
func handleVerifyError(err error) {
	errMsg := err.Error()

	switch {
	case containsAny(errMsg, "expired"):
		fmt.Println("❌ Token 已过期，请重新登录")

	case containsAny(errMsg, "kid", "not found"):
		fmt.Println("❌ 签名密钥未找到，可能是旧的 Token")

	case containsAny(errMsg, "signature"):
		fmt.Println("❌ Token 签名无效")

	case containsAny(errMsg, "audience"):
		fmt.Println("❌ Token 的 audience 不匹配")

	case containsAny(errMsg, "issuer"):
		fmt.Println("❌ Token 的 issuer 不匹配")

	default:
		fmt.Printf("❌ 验证失败: %v\n", err)
	}
}

// containsAny 检查字符串是否包含任意一个子串
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// 示例 8: 并发验证
func example8_ConcurrentVerification() {
	fmt.Println("\n=== 示例 8: 并发验证 ===")

	cfg := authnsdk.Config{
		JWKSURL: "https://iam.example.com/.well-known/jwks.json",
	}

	verifier, err := authnsdk.NewVerifier(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

	// 并发验证 100 个请求
	const numRequests = 100
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			_, err := verifier.Verify(ctx, token, nil)
			results <- err
		}(i)
	}

	// 收集结果
	successCount := 0
	failCount := 0

	for i := 0; i < numRequests; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Printf("并发验证完成:\n")
	fmt.Printf("  - 成功: %d\n", successCount)
	fmt.Printf("  - 失败: %d\n", failCount)
}

// main 主函数，可以取消注释需要运行的示例
func main() {
	fmt.Println("IAM AuthN SDK - 基础使用示例")
	fmt.Println("================================")

	// 运行各个示例（根据需要取消注释）
	// example1_BasicLocalVerification()
	// example2_WithAudienceIssuer()
	// example3_CustomJWKSConfig()
	// example4_RemoteVerification()
	// example5_ForceRemoteVerification()
	// example6_CustomAttributes()
	// example7_ErrorHandling()
	// example8_ConcurrentVerification()

	fmt.Println("\n提示: 请取消注释 main 函数中的示例代码以运行相应示例")
}
