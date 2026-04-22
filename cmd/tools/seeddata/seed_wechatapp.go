package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/idp/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/container"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
)

// ==================== 微信应用相关类型定义 ====================

// wechatAppRecord 微信应用种子数据
type wechatAppRecord struct {
	Alias     string
	AppID     string
	Name      string
	Type      string // MiniProgram/MP
	Status    string
	AppSecret string // 可选，用于设置
}

// ==================== 微信应用 Seed 函数 ====================

// seedWechatApps 创建微信应用数据
//
// 业务说明：
// - 微信应用用于小程序或公众号的身份提供商集成
// - 从配置文件读取微信应用数据
// - 使用应用服务进行创建和查询，遵循领域驱动设计
// - AppSecret 通过应用服务进行加密存储
//
// 幂等性：先查询是否存在，不存在则创建，存在则跳过（或更新凭据）
func seedWechatApps(ctx context.Context, deps *dependencies) error {
	config := deps.Config
	if config == nil || len(config.WechatApps) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有微信应用数据，跳过")
		return nil
	}

	deps.Logger.Infow("📋 开始创建微信应用数据...", "count", len(config.WechatApps))

	encryptionKey, err := resolveWechatAppEncryptionKey(config)
	if err != nil {
		return err
	}
	if len(encryptionKey) == 32 {
		deps.Logger.Debugw("🔐 使用配置文件中的加密密钥", "key_length", len(encryptionKey))
	} else {
		deps.Logger.Warnw("⚠️  未配置加密密钥，当前仅同步微信应用元数据，不应写入任何密钥类凭据")
	}

	// 初始化容器和 IDP 模块（传递加密密钥）
	// Redis: 用于 IDP 模块的 Access Token 缓存和 Authn 模块的 Token 存储
	// EventBus: nil（种子数据工具不需要消息队列）
	c := container.NewContainer(deps.DB, deps.RedisCache, nil, encryptionKey)
	if err := c.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize container: %w", err)
	}

	if c.IDPModule == nil {
		return fmt.Errorf("IDP module not initialized")
	}

	appService := c.IDPModule.WechatAppService
	credentialService := c.IDPModule.WechatAppCredentialService

	// 从配置读取微信应用
	var apps []wechatAppRecord
	for _, wa := range config.WechatApps {
		apps = append(apps, wechatAppRecord{
			Alias:     wa.Alias,
			AppID:     wa.AppID,
			Name:      wa.Name,
			Type:      wa.Type,
			Status:    wa.Status,
			AppSecret: wa.AppSecret,
		})
	}

	for _, app := range apps {
		// 转换应用类型
		var appType domain.AppType
		switch app.Type {
		case "MiniProgram":
			appType = domain.MiniProgram
		case "MP":
			appType = domain.MP
		default:
			deps.Logger.Warnw("⚠️  未知的应用类型，跳过",
				"app_id", app.AppID,
				"type", app.Type)
			continue
		}

		// 先查询应用是否已存在
		existingApp, err := appService.GetApp(ctx, app.AppID)
		if err == nil && existingApp != nil {
			deps.Logger.Debugw("ℹ️  微信应用已存在，跳过创建",
				"app_id", app.AppID,
				"name", existingApp.Name)

			// 如果提供了 AppSecret，更新凭据
			if app.AppSecret != "" {
				if err := credentialService.RotateAuthSecret(ctx, app.AppID, app.AppSecret); err != nil {
					deps.Logger.Warnw("⚠️  更新 AppSecret 失败",
						"app_id", app.AppID,
						"error", err)
				} else {
					deps.Logger.Debugw("✅ AppSecret 已更新",
						"app_id", app.AppID)
				}
			}
			continue
		}

		// 创建新应用
		createDTO := wechatapp.CreateWechatAppDTO{
			AppID:     app.AppID,
			Name:      app.Name,
			Type:      appType,
			AppSecret: app.AppSecret,
		}

		result, err := appService.CreateApp(ctx, createDTO)
		if err != nil {
			return fmt.Errorf("failed to create wechat app %s: %w", app.AppID, err)
		}

		deps.Logger.Infow("✅ 微信应用已创建",
			"app_id", result.AppID,
			"name", result.Name,
			"type", result.Type,
			"status", result.Status)
	}

	deps.Logger.Infow("✅ 微信应用数据创建完成", "count", len(apps))
	return nil
}

func resolveWechatAppEncryptionKey(config *SeedConfig) ([]byte, error) {
	if config == nil {
		return nil, nil
	}

	rawKey := strings.TrimSpace(config.EncryptionKey)
	if rawKey == "" {
		for _, app := range config.WechatApps {
			if strings.TrimSpace(app.AppSecret) != "" {
				return nil, fmt.Errorf("encryption_key is required when wechat_apps[%s].app_secret is configured", app.Alias)
			}
		}
		return nil, nil
	}

	if decoded, err := base64.StdEncoding.DecodeString(rawKey); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if len(rawKey) == 32 {
		return []byte(rawKey), nil
	}

	return nil, fmt.Errorf("invalid encryption key: must be 32 bytes or base64-encoded 32 bytes")
}
