package redis

import (
	"fmt"

	rediskeyspace "github.com/FangcunMount/component-base/pkg/redis/keyspace"
)

var (
	refreshTokenKeyspace          = rediskeyspace.New("refresh_token")
	tokenBlacklistKeyspace        = rediskeyspace.New("token_blacklist")
	otpKeyspace                   = rediskeyspace.New("otp")
	otpSendGateKeyspace           = otpKeyspace.Child("sendgate")
	wechatAccessTokenKeyspace     = rediskeyspace.New("idp").Child("wechat").Child("token")
	wechatAccessTokenLockKeyspace = wechatAccessTokenKeyspace.Child("lock")
)

func refreshTokenRedisKey(tokenValue string) string {
	return refreshTokenKeyspace.Prefix(tokenValue)
}

func tokenBlacklistRedisKey(tokenID string) string {
	return tokenBlacklistKeyspace.Prefix(tokenID)
}

func otpRedisKey(phoneE164, scene, code string) string {
	return otpKeyspace.Prefix(fmt.Sprintf("%s:%s:%s", scene, phoneE164, code))
}

func otpSendGateRedisKey(phoneE164, scene string) string {
	return otpSendGateKeyspace.Prefix(fmt.Sprintf("%s:%s", scene, phoneE164))
}

func wechatAccessTokenRedisKey(appID string) string {
	return wechatAccessTokenKeyspace.Prefix(appID)
}

func wechatAccessTokenLockRedisKey(appID string) string {
	return wechatAccessTokenLockKeyspace.Prefix(appID)
}
