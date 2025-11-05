package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
)

// OTPVerifierImpl OTP验证器的Redis实现
type OTPVerifierImpl struct {
	client *redis.Client
}

// 确保实现了接口
var _ port.OTPVerifier = (*OTPVerifierImpl)(nil)

// NewOTPVerifier 创建OTP验证器
func NewOTPVerifier(client *redis.Client) port.OTPVerifier {
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
		// Redis错误，返回验证失败
		return false
	}

	return result == 1
}
