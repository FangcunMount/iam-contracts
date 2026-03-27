package loginprep

import "context"

// LoginPreparationService 登录预准备：在调用 Login 签发令牌之前完成侧车动作。
// 例如：手机验证码发码、（未来）微信网站扫码创建会话并返回二维码等。
// 与 LoginApplicationService 解耦，避免把「认证+颁票」与「渠道预准备」绑在同一端口。
type LoginPreparationService interface {
	// SendPhoneOTPForLogin 发送登录用手机短信验证码（写入 Redis，场景 login，与领域 OTP 校验一致）
	SendPhoneOTPForLogin(ctx context.Context, phone string) error
	// 后续可扩展：CreateWeChatWebScanSession(ctx, appID) (ticketURL, pollToken, error) 等
}
