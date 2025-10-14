package port

import "context"

// WeChatAuthPort 微信认证端口
//
// 用于与微信服务端交互，换取用户的 openID
type WeChatAuthPort interface {
	// ExchangeOpenID 通过微信授权码换取 openID
	//
	// 参数:
	//   - code: 微信授权码
	//   - appID: 微信应用 ID
	//
	// 返回:
	//   - openID: 微信用户的 openID
	//   - err: 错误信息（如网络错误、code 无效等）
	ExchangeOpenID(ctx context.Context, code, appID string) (openID string, err error)
}
