package sms

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/logger"
)

// LogSender 将登录 OTP 打到日志（仅用于开发/联调，禁止在生产依赖）
type LogSender struct{}

// SendLoginOTP 记录验证码，不调用真实短信网关
func (LogSender) SendLoginOTP(ctx context.Context, phoneE164, code string) error {
	logger.L(ctx).Infow("sms login otp",
		"phone", phoneE164,
		"code", code,
	)
	return nil
}
