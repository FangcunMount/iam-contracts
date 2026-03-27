package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
)

// OTPVerifierImpl OTP验证器的Redis实现
type OTPVerifierImpl struct {
	client *redis.Client
}

// 确保实现了接口
var (
	_ authentication.OTPVerifier   = (*OTPVerifierImpl)(nil)
	_ authentication.OTPCodeStore = (*OTPVerifierImpl)(nil)
	_ authentication.OTPSendGate  = (*OTPVerifierImpl)(nil)
)

// NewOTPVerifier 创建 OTP Redis 适配器（验证、写入、发送频控共用同一实现）
func NewOTPVerifier(client *redis.Client) *OTPVerifierImpl {
	return &OTPVerifierImpl{client: client}
}

// VerifyAndConsume 验证OTP并标记为已使用（原子操作，防止重放攻击）
// phoneE164: E164格式的手机号，如 +8613800138000
// scene: OTP使用场景，如 "login", "register", "reset_password"
// code: 验证码
func (v *OTPVerifierImpl) VerifyAndConsume(ctx context.Context, phoneE164, scene, code string) bool {
	// Redis key格式: otp:{scene}:{phone}:{code}
	// 例如: otp:login:+8613800138000:123456
	key := otpRedisKey(phoneE164, scene, code)

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
		// Redis Hook 已经记录了 EVAL 命令错误，只返回错误即可
		return false
	}

	if result == 1 {
		redisInfo(ctx, "OTP verified",
			log.String("scene", scene),
			log.String("phone", phoneE164),
		)
	}
	// Redis Hook 已经记录了 EVAL 命令执行，不需要记录 OTP not found

	return result == 1
}

func otpRedisKey(phoneE164, scene, code string) string {
	return fmt.Sprintf("otp:%s:%s:%s", scene, phoneE164, code)
}

func otpSendGateKey(phoneE164, scene string) string {
	return fmt.Sprintf("otp:sendgate:%s:%s", scene, phoneE164)
}

// Put 写入待校验 OTP，与 VerifyAndConsume 使用同一 key 规则。
func (v *OTPVerifierImpl) Put(ctx context.Context, phoneE164, scene, code string, ttl time.Duration) error {
	key := otpRedisKey(phoneE164, scene, code)
	return v.client.Set(ctx, key, "1", ttl).Err()
}

// Delete 删除 OTP 键（短信发送失败时回滚）。
func (v *OTPVerifierImpl) Delete(ctx context.Context, phoneE164, scene, code string) error {
	key := otpRedisKey(phoneE164, scene, code)
	return v.client.Del(ctx, key).Err()
}

// TryAcquire 使用 SET NX 实现发送冷却窗口。
func (v *OTPVerifierImpl) TryAcquire(ctx context.Context, phoneE164, scene string, cooldown time.Duration) (bool, error) {
	key := otpSendGateKey(phoneE164, scene)
	ok, err := v.client.SetNX(ctx, key, "1", cooldown).Result()
	if err != nil {
		return false, fmt.Errorf("otp send gate: %w", err)
	}
	return ok, nil
}
