package main

import "strings"

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
