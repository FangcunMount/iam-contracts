// Token 验证示例
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
)

func main() {
	ctx := context.Background()

	// 创建客户端
	client, err := sdk.NewClient(ctx, &sdk.Config{
		Endpoint: "iam.example.com:8081",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 方式1：创建本地 Token 验证器（推荐高频场景）
	verifier, err := sdk.NewTokenVerifier(
		&sdk.TokenVerifyConfig{
			AllowedAudience: []string{"my-app"},
			AllowedIssuer:   "https://iam.example.com",
			ClockSkew:       5 * time.Minute,
		},
		&sdk.JWKSConfig{
			URL:             "https://iam.example.com/.well-known/jwks.json",
			RefreshInterval: 5 * time.Minute,
			RequestTimeout:  5 * time.Second,
			FallbackOnError: true,
		},
		client, // 可选：用于远程验证降级
	)
	if err != nil {
		log.Fatalf("创建验证器失败: %v", err)
	}

	// 验证 Token
	token := "eyJhbGciOiJSUzI1NiIs..."
	result, err := verifier.Verify(ctx, token, nil)
	if err != nil {
		log.Printf("验证失败: %v", err)
		return
	}

	if result.Valid {
		fmt.Printf("Token 有效\n")
		fmt.Printf("用户 ID: %s\n", result.Claims.UserID)
		fmt.Printf("租户 ID: %s\n", result.Claims.TenantID)
		fmt.Printf("过期时间: %s\n", result.Claims.ExpiresAt)
	}
}
