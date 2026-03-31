package main

import (
	"strings"
)

// accountOperaExternalID 返回运营账号写入 auth_accounts.external_id 的登录标识。
// external_id 是主配置；username 仅作为兼容旧 seeddata 的回退字段。
func accountOperaExternalID(ac AccountConfig, userEmail string) string {
	if v := strings.TrimSpace(ac.ExternalID); v != "" {
		return v
	}
	if v := strings.TrimSpace(ac.Username); v != "" {
		return v
	}
	return strings.TrimSpace(userEmail)
}
