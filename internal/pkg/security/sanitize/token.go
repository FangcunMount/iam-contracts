package sanitize

import (
	"fmt"
	"strings"
)

const (
	tokenPrefixLen = 6
	tokenSuffixLen = 4
)

// MaskToken 返回令牌的脱敏信息，避免在日志中泄露完整凭证。
func MaskToken(token string) string {
	if token == "" {
		return ""
	}

	length := len(token)
	if length <= tokenPrefixLen+tokenSuffixLen {
		return fmt.Sprintf("%s(len=%d)", strings.Repeat("*", length), length)
	}

	prefix := token[:tokenPrefixLen]
	suffix := token[length-tokenSuffixLen:]
	return fmt.Sprintf("%s***%s(len=%d)", prefix, suffix, length)
}
