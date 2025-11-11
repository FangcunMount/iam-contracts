package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
)

// OTPVerifierImpl OTP验证器的Redis实现
type OTPVerifierImpl struct {
	client *redis.Client
}

// 确保实现了接口
var _ authentication.OTPVerifier = (*OTPVerifierImpl)(nil)

// NewOTPVerifier 创建OTP验证器
func NewOTPVerifier(client *redis.Client) authentication.OTPVerifier {
	return &OTPVerifierImpl{client: client}
}

// VerifyAndConsume 验证OTP并标记为已使用（原子操作，防止重放攻击）
// phoneE164: E164格式的手机号，如 +8613800138000
// scene: OTP使用场景，如 "login", "register", "reset_password"
// code: 验证码
func (v *OTPVerifierImpl) VerifyAndConsume(ctx context.Context, phoneE164, scene, code string) bool {
	// Redis key格式: otp:{scene}:{phone}:{code}
	// 例如: otp:login:+8613800138000:123456
	key := fmt.Sprintf("otp:%s:%s:%s", scene, phoneE164, code)

	// 使用Lua脚本实现原子性的验证+删除操作
	// 这样可以防止同一个验证码被多次使用（重放攻击）
	script := `
		if redis.call("exists", KEYS[1]) == 1 then
			redis.call("del", KEYS[1])
			return 1
		else
			return 0
		end
	`

	result, err := v.client.Eval(ctx, script, []string{key}).Int()
	if err != nil {
		redisError(ctx, "OTP verification failed",
			log.String("error", err.Error()),
			log.String("key", key),
			log.String("scene", scene),
		)
		return false
	}

	if result == 1 {
		redisInfo(ctx, "OTP verified",
			log.String("scene", scene),
			log.String("phone", phoneE164),
		)
	} else {
		redisDebug(ctx, "OTP not found or already consumed",
			log.String("scene", scene),
			log.String("phone", phoneE164),
		)
	}

	return result == 1
}
