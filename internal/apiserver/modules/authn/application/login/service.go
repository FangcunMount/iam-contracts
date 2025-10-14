// Package login 登录应用服务
package login

import (
	"context"

	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	authService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
)

// LoginService 登录应用服务
type LoginService struct {
	authService  *authService.AuthenticationService // 认证服务
	tokenService *authService.TokenService          // 令牌服务
}

// NewLoginService 创建登录应用服务
func NewLoginService(
	authService *authService.AuthenticationService,
	tokenService *authService.TokenService,
) *LoginService {
	return &LoginService{
		authService:  authService,
		tokenService: tokenService,
	}
}

// LoginWithPasswordRequest 密码登录请求
type LoginWithPasswordRequest struct {
	Username string
	Password string
	IP       string // 客户端IP（可选）
	Device   string // 设备信息（可选）
}

// LoginWithPasswordResponse 密码登录响应
type LoginWithPasswordResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // 秒
}

// LoginWithPassword 用户名密码登录
func (s *LoginService) LoginWithPassword(ctx context.Context, req *LoginWithPasswordRequest) (*LoginWithPasswordResponse, error) {
	if req == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "request is required")
	}

	// 1. 创建用户名密码凭证
	credential := authentication.NewUsernamePasswordCredential(req.Username, req.Password)

	// 2. 执行认证
	auth, err := s.authService.Authenticate(ctx, credential)
	if err != nil {
		return nil, err
	}

	// 3. 添加认证元数据
	if req.IP != "" {
		auth.WithMetadata("ip", req.IP)
	}
	if req.Device != "" {
		auth.WithMetadata("device", req.Device)
	}

	// 4. 颁发令牌
	tokenPair, err := s.tokenService.IssueToken(ctx, auth)
	if err != nil {
		return nil, err
	}

	// 5. 构造响应
	return &LoginWithPasswordResponse{
		AccessToken:  tokenPair.AccessToken.Value,
		RefreshToken: tokenPair.RefreshToken.Value,
		TokenType:    "Bearer",
		ExpiresIn:    int64(tokenPair.AccessToken.RemainingDuration().Seconds()),
	}, nil
}

// LoginWithWeChatRequest 微信登录请求
type LoginWithWeChatRequest struct {
	Code   string // 微信授权码
	AppID  string // 微信应用ID
	IP     string // 客户端IP（可选）
	Device string // 设备信息（可选）
}

// LoginWithWeChatResponse 微信登录响应
type LoginWithWeChatResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // 秒
}

// LoginWithWeChat 微信登录
func (s *LoginService) LoginWithWeChat(ctx context.Context, req *LoginWithWeChatRequest) (*LoginWithWeChatResponse, error) {
	if req == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "request is required")
	}

	// 1. 创建微信凭证
	credential := authentication.NewWeChatCodeCredential(req.Code, req.AppID)

	// 2. 执行认证
	auth, err := s.authService.Authenticate(ctx, credential)
	if err != nil {
		return nil, err
	}

	// 3. 添加认证元数据
	if req.IP != "" {
		auth.WithMetadata("ip", req.IP)
	}
	if req.Device != "" {
		auth.WithMetadata("device", req.Device)
	}

	// 4. 颁发令牌
	tokenPair, err := s.tokenService.IssueToken(ctx, auth)
	if err != nil {
		return nil, err
	}

	// 5. 构造响应
	return &LoginWithWeChatResponse{
		AccessToken:  tokenPair.AccessToken.Value,
		RefreshToken: tokenPair.RefreshToken.Value,
		TokenType:    "Bearer",
		ExpiresIn:    int64(tokenPair.AccessToken.RemainingDuration().Seconds()),
	}, nil
}
