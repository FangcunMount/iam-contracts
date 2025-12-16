package main

import (
	"context"
	"strings"
	"sync"
	"time"
)

// ==================== 辅助函数 ====================

// genderStringToUint8 将字符串性别转换为 uint8
func genderStringToUint8(gender string) uint8 {
	switch gender {
	case "male":
		return 1
	case "female":
		return 2
	default:
		return 0
	}
}

// normalizeLoginID 尝试将手机号补全为 E.164，避免账号 ExternalID=手机号时登录失败
func normalizeLoginID(loginID string) string {
	if loginID == "" {
		return loginID
	}
	trimmed := strings.TrimSpace(loginID)
	if strings.HasPrefix(trimmed, "+") {
		return trimmed
	}
	// 简单规则：11位国内手机号前补 +86
	if len(trimmed) == 11 && strings.HasPrefix(trimmed, "1") {
		return "+86" + trimmed
	}
	return trimmed
}

// isPhoneLike 粗略判断是否是手机号/E.164
func isPhoneLike(id string) bool {
	if id == "" {
		return false
	}
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "+") {
		return true
	}
	if len(id) == 11 && strings.HasPrefix(id, "1") {
		return true
	}
	return false
}

// resolveLoginID 为 operation 账号选择登录标识：
// 1) external_id/username 若像手机号则补全 E.164
// 2) 否则使用关联用户手机号
// 3) 最后回退 external_id/username
func resolveLoginID(ac AccountConfig, uc UserConfig) string {
	if isPhoneLike(ac.ExternalID) {
		return normalizeLoginID(ac.ExternalID)
	}
	if isPhoneLike(ac.Username) {
		return normalizeLoginID(ac.Username)
	}
	if isPhoneLike(uc.Phone) {
		return normalizeLoginID(uc.Phone)
	}
	if ac.ExternalID != "" {
		return ac.ExternalID
	}
	if ac.Username != "" {
		return ac.Username
	}
	if uc.Phone != "" {
		return normalizeLoginID(uc.Phone)
	}
	return ""
}

// superAdmin token 缓存
var (
	superAdminToken       string
	superAdminTokenExpiry time.Time
	superAdminTokenMu     sync.Mutex
)

// getSuperAdminToken 返回缓存的超级管理员 token，如过期则重新登录获取并缓存
func getSuperAdminToken(ctx context.Context, iamServiceURL, loginID, password string) (string, error) {
	superAdminTokenMu.Lock()
	// 返回前先检查缓存
	if superAdminToken != "" && time.Now().Before(superAdminTokenExpiry) {
		token := superAdminToken
		superAdminTokenMu.Unlock()
		return token, nil
	}
	superAdminTokenMu.Unlock()

	// 未命中缓存或已过期，执行登录获取 TokenPair
	tp, err := loginAsSuperAdmin(ctx, iamServiceURL, loginID, password)
	if err != nil {
		return "", err
	}

	// loginAsSuperAdmin 返回 TokenPair (updated signature)
	token := tp.AccessToken
	// 计算过期时间，若服务未返回 expires_in 则默认 10 分钟
	var expiry time.Time
	if tp.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(tp.ExpiresIn) * time.Second)
	} else {
		expiry = time.Now().Add(10 * time.Minute)
	}
	// 提前 30 秒刷新
	expiry = expiry.Add(-30 * time.Second)

	superAdminTokenMu.Lock()
	superAdminToken = token
	superAdminTokenExpiry = expiry
	superAdminTokenMu.Unlock()

	return token, nil
}
