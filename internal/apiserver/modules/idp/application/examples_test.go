package application_test

import (
	"context"
	"fmt"
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatsession"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
)

// 本文件展示如何使用 IDP 模块的应用服务层

// Example_createWechatApp 示例：创建微信应用
func Example_createWechatApp() {
	// 假设已经通过依赖注入容器获取了应用服务
	var appServices *application.ApplicationServices

	ctx := context.Background()

	// 1. 准备创建微信应用的 DTO
	dto := wechatapp.CreateWechatAppDTO{
		AppID:     "wx1234567890abcdef",
		Name:      "我的测试小程序",
		Type:      domain.MiniProgram,
		AppSecret: "your-app-secret-here-32-chars",
	}

	// 2. 调用应用服务创建微信应用
	result, err := appServices.WechatApp.CreateApp(ctx, dto)
	if err != nil {
		fmt.Printf("创建失败: %v\n", err)
		return
	}

	// 3. 使用返回的结果
	fmt.Printf("创建成功:\n")
	fmt.Printf("  内部 ID: %s\n", result.ID)
	fmt.Printf("  微信 AppID: %s\n", result.AppID)
	fmt.Printf("  应用名称: %s\n", result.Name)
	fmt.Printf("  应用类型: %s\n", result.Type)
	fmt.Printf("  应用状态: %s\n", result.Status)
}

// Example_getWechatApp 示例：查询微信应用
func Example_getWechatApp() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// 查询微信应用
	result, err := appServices.WechatApp.GetApp(ctx, "wx1234567890abcdef")
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}

	fmt.Printf("查询成功: %s (%s)\n", result.Name, result.AppID)
}

// Example_rotateAuthSecret 示例：轮换认证密钥
func Example_rotateAuthSecret() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// 轮换 AppSecret
	err := appServices.WechatAppCredential.RotateAuthSecret(
		ctx,
		"wx1234567890abcdef",
		"new-app-secret-here-32-chars-xx",
	)
	if err != nil {
		fmt.Printf("轮换失败: %v\n", err)
		return
	}

	fmt.Println("AppSecret 轮换成功")
}

// Example_rotateMsgSecret 示例：轮换消息加解密密钥
func Example_rotateMsgSecret() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// 轮换消息加解密密钥
	err := appServices.WechatAppCredential.RotateMsgSecret(
		ctx,
		"wx1234567890abcdef",
		"callback-token-here",
		"encoding-aes-key-43-chars-base64-encoded",
	)
	if err != nil {
		fmt.Printf("轮换失败: %v\n", err)
		return
	}

	fmt.Println("消息加解密密钥轮换成功")
}

// Example_getAccessToken 示例：获取访问令牌（自动缓存和刷新）
func Example_getAccessToken() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// 获取访问令牌（自动处理缓存、过期检查、单飞刷新）
	token, err := appServices.WechatAppToken.GetAccessToken(ctx, "wx1234567890abcdef")
	if err != nil {
		fmt.Printf("获取失败: %v\n", err)
		return
	}

	fmt.Printf("访问令牌: %s\n", token)

	// 使用令牌调用微信 API
	// ...
}

// Example_refreshAccessToken 示例：强制刷新访问令牌
func Example_refreshAccessToken() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// 强制刷新访问令牌（不使用缓存）
	token, err := appServices.WechatAppToken.RefreshAccessToken(ctx, "wx1234567890abcdef")
	if err != nil {
		fmt.Printf("刷新失败: %v\n", err)
		return
	}

	fmt.Printf("新的访问令牌: %s\n", token)
}

// Example_wechatLogin 示例：微信小程序登录
func Example_wechatLogin() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// 1. 准备登录 DTO（通常从前端传来）
	dto := wechatsession.LoginWithCodeDTO{
		AppID:  "wx1234567890abcdef",
		JSCode: "071AbcDef123456789", // 前端调用 wx.login() 获取的 code
	}

	// 2. 调用应用服务进行登录
	result, err := appServices.WechatAuth.LoginWithCode(ctx, dto)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		return
	}

	// 3. 使用登录结果
	fmt.Printf("登录成功:\n")
	fmt.Printf("  身份提供商: %s\n", result.Provider)
	fmt.Printf("  OpenID: %s\n", result.OpenID)
	if result.UnionID != nil {
		fmt.Printf("  UnionID: %s\n", *result.UnionID)
	}
	if result.DisplayName != nil {
		fmt.Printf("  昵称: %s\n", *result.DisplayName)
	}
	fmt.Printf("  会话版本: %d\n", result.Version)
	fmt.Printf("  过期时间: %d 秒\n", result.ExpiresInSec)

	// 4. 后续业务逻辑
	// - 根据 OpenID 查找或创建用户
	// - 生成业务系统的 JWT Token
	// - 返回给前端
}

// Example_decryptUserPhone 示例：解密用户手机号
func Example_decryptUserPhone() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// 1. 准备解密 DTO（通常从前端传来）
	dto := wechatsession.DecryptPhoneDTO{
		AppID:         "wx1234567890abcdef",
		OpenID:        "oABC123456789",
		EncryptedData: "encrypted-phone-data-from-wx", // 前端调用 getPhoneNumber 获取
		IV:            "iv-from-wx",
	}

	// 2. 调用应用服务解密手机号
	phone, err := appServices.WechatAuth.DecryptUserPhone(ctx, dto)
	if err != nil {
		fmt.Printf("解密失败: %v\n", err)
		return
	}

	// 3. 使用解密后的手机号
	fmt.Printf("用户手机号: %s\n", phone)

	// 4. 后续业务逻辑
	// - 更新用户的手机号
	// - 发送验证码
	// - 绑定账号
}

// Example_completeWechatLoginFlow 示例：完整的微信登录业务流程
func Example_completeWechatLoginFlow() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// ========== 步骤 1: 微信小程序登录 ==========
	loginDTO := wechatsession.LoginWithCodeDTO{
		AppID:  "wx1234567890abcdef",
		JSCode: "071AbcDef123456789",
	}

	loginResult, err := appServices.WechatAuth.LoginWithCode(ctx, loginDTO)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		return
	}

	fmt.Printf("步骤 1: 微信登录成功，OpenID=%s\n", loginResult.OpenID)

	// ========== 步骤 2: 根据 OpenID 查找或创建用户 ==========
	// 这里调用 UC 模块的应用服务
	// userService := ucAppServices.UserQuery
	// user, err := userService.GetByExternalID(ctx, "wechat_miniprogram", loginResult.OpenID)
	// if err != nil {
	// 	  // 用户不存在，创建新用户
	//    registerDTO := uc.RegisterUserDTO{
	//        Name:  *loginResult.DisplayName,
	//        Phone: *loginResult.Phone,
	//    }
	//    user, err = userService.Register(ctx, registerDTO)
	// }

	fmt.Println("步骤 2: 用户查找/创建完成")

	// ========== 步骤 3: 生成业务系统的 JWT Token ==========
	// 这里调用 AUTHN 模块的应用服务
	// authnService := authnAppServices.TokenManagement
	// tokenDTO := authn.GenerateTokenDTO{
	//    UserID:    user.ID,
	//    AccountID: account.ID,
	//    Type:      authn.AccessToken,
	// }
	// businessToken, err := authnService.GenerateToken(ctx, tokenDTO)

	fmt.Println("步骤 3: 业务 Token 生成完成")

	// ========== 步骤 4: 返回给前端 ==========
	// response := LoginResponse{
	//    Token:     businessToken.Token,
	//    ExpiresIn: businessToken.ExpiresIn,
	//    User: UserInfo{
	//        ID:       user.ID,
	//        Name:     user.Name,
	//        Avatar:   *loginResult.AvatarURL,
	//    },
	// }

	fmt.Println("步骤 4: 登录流程完成，返回给前端")
}

// Example_tokenCaching 示例：访问令牌缓存机制
func Example_tokenCaching() {
	var appServices *application.ApplicationServices
	ctx := context.Background()
	appID := "wx1234567890abcdef"

	// 第一次调用：从微信 API 获取（缓存 Miss）
	start := time.Now()
	token1, _ := appServices.WechatAppToken.GetAccessToken(ctx, appID)
	duration1 := time.Since(start)
	fmt.Printf("第 1 次调用 (缓存 Miss): %s (耗时: %v)\n", token1[:20]+"...", duration1)

	// 第二次调用：从缓存获取（缓存 Hit）
	start = time.Now()
	token2, _ := appServices.WechatAppToken.GetAccessToken(ctx, appID)
	duration2 := time.Since(start)
	fmt.Printf("第 2 次调用 (缓存 Hit): %s (耗时: %v)\n", token2[:20]+"...", duration2)

	// 第二次调用应该明显更快
	fmt.Printf("性能提升: %.2fx\n", float64(duration1)/float64(duration2))

	// 强制刷新
	token3, _ := appServices.WechatAppToken.RefreshAccessToken(ctx, appID)
	fmt.Printf("强制刷新后: %s\n", token3[:20]+"...")
}

// Example_credentialRotation 示例：定期轮换凭据（安全最佳实践）
func Example_credentialRotation() {
	var appServices *application.ApplicationServices
	ctx := context.Background()
	appID := "wx1234567890abcdef"

	// ========== 场景 1: 定期轮换 AppSecret（每 90 天） ==========
	// 这通常在定时任务中执行
	newAppSecret := generateNewAppSecret() // 生成新的 AppSecret
	err := appServices.WechatAppCredential.RotateAuthSecret(ctx, appID, newAppSecret)
	if err != nil {
		fmt.Printf("轮换 AppSecret 失败: %v\n", err)
		return
	}
	fmt.Println("定期轮换 AppSecret 成功（每 90 天）")

	// ========== 场景 2: 应急轮换（密钥泄露） ==========
	// 当发现密钥泄露时，立即轮换
	emergencyAppSecret := generateNewAppSecret()
	err = appServices.WechatAppCredential.RotateAuthSecret(ctx, appID, emergencyAppSecret)
	if err != nil {
		fmt.Printf("应急轮换 AppSecret 失败: %v\n", err)
		return
	}
	fmt.Println("应急轮换 AppSecret 成功（密钥泄露）")

	// 轮换后，下次获取访问令牌会使用新密钥
	token, _ := appServices.WechatAppToken.RefreshAccessToken(ctx, appID)
	fmt.Printf("使用新密钥获取的 Token: %s\n", token[:20]+"...")
}

// Example_errorHandling 示例：错误处理
func Example_errorHandling() {
	var appServices *application.ApplicationServices
	ctx := context.Background()

	// ========== 错误 1: 应用不存在 ==========
	_, err := appServices.WechatApp.GetApp(ctx, "wx_not_exists")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		// 输出: 错误: wechat app not found: wx_not_exists
	}

	// ========== 错误 2: 参数验证失败 ==========
	dto := wechatapp.CreateWechatAppDTO{
		AppID: "", // AppID 为空
		Name:  "测试应用",
		Type:  domain.MiniProgram,
	}
	_, err = appServices.WechatApp.CreateApp(ctx, dto)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		// 输出: 错误: appID cannot be empty
	}

	// ========== 错误 3: 业务规则违反（重复创建） ==========
	dto2 := wechatapp.CreateWechatAppDTO{
		AppID: "wx_already_exists",
		Name:  "测试应用",
		Type:  domain.MiniProgram,
	}
	_, err = appServices.WechatApp.CreateApp(ctx, dto2)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		// 输出: 错误: wechat app with the given appID already exists
	}

	// ========== 错误 4: 微信 API 调用失败 ==========
	loginDTO := wechatsession.LoginWithCodeDTO{
		AppID:  "wx1234567890abcdef",
		JSCode: "invalid_code", // 无效的 code
	}
	_, err = appServices.WechatAuth.LoginWithCode(ctx, loginDTO)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		// 输出: 错误: failed to login with code: invalid jsCode
	}
}

// 辅助函数
func generateNewAppSecret() string {
	// 实际应该使用加密安全的随机数生成器
	return "new-secure-app-secret-32-chars"
}
