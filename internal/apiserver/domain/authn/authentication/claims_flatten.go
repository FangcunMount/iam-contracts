package authentication

import (
	"encoding/json"
	"fmt"
)

// 与 JWT 标准 claim 及 CustomClaims 顶层字段冲突的键，不得写入 attributes。
var jwtReservedClaimKeys = map[string]struct{}{
	"token_type": {}, "user_id": {}, "account_id": {}, "tenant_id": {},
	"jti": {}, "sub": {}, "iss": {}, "aud": {}, "exp": {}, "iat": {}, "nbf": {},
	"amr": {}, "attributes": {}, "audience": {},
}

// FlattenClaimsForJWT 将 Principal.Claims 转为可写入 JWT `attributes` 的字符串 map。
func FlattenClaimsForJWT(m map[string]any) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if _, reserved := jwtReservedClaimKeys[k]; reserved {
			continue
		}
		if v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			out[k] = t
		case fmt.Stringer:
			out[k] = t.String()
		default:
			if b, err := json.Marshal(t); err == nil {
				out[k] = string(b)
			} else {
				out[k] = fmt.Sprint(v)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ClaimsFromStringMap 将刷新令牌里持久化的字符串 map 还原为 Principal.Claims。
func ClaimsFromStringMap(m map[string]string) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
