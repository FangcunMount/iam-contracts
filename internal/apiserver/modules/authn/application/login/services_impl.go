package login

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	authService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service"
	tokenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/token/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

type loginApplicationService struct {
	strategyFactory *authService.StrategyFactory
	tokenIssuer     tokenPort.TokenIssuer
	tokenRefresher  tokenPort.TokenRefresher
}

var _ LoginApplicationService = (*loginApplicationService)(nil)

func NewLoginApplicationService(
	strategyFactory *authService.StrategyFactory,
	tokenIssuer tokenPort.TokenIssuer,
	tokenRefresher tokenPort.TokenRefresher,
) LoginApplicationService {
	return &loginApplicationService{
		strategyFactory: strategyFactory,
		tokenIssuer:     tokenIssuer,
		tokenRefresher:  tokenRefresher,
	}
}

func (s *loginApplicationService) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	scenario, authInput, err := s.prepareAuthentication(req)
	if err != nil {
		return nil, err
	}

	strategy := s.strategyFactory.CreateStrategy(scenario)
	if strategy == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "authentication strategy not available for type: %s", req.AuthType)
	}

	decision, err := strategy.Authenticate(ctx, authInput)
	if err != nil {
		return nil, err
	}

	if !decision.OK {
		return nil, s.convertAuthError(decision.ErrCode)
	}

	tokenPair, err := s.tokenIssuer.IssueToken(ctx, decision.Principal)
	if err != nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "failed to issue token: %v", err)
	}

	return &LoginResult{
		Principal: decision.Principal,
		TokenPair: tokenPair,
		UserID:    decision.Principal.UserID,
		AccountID: decision.Principal.AccountID,
		TenantID:  decision.Principal.TenantID,
	}, nil
}

// Logout 登出接口 - 撤销令牌
func (s *loginApplicationService) Logout(ctx context.Context, req LogoutRequest) error {
	// 至少需要提供一个令牌
	if (req.AccessToken == nil || *req.AccessToken == "") &&
		(req.RefreshToken == nil || *req.RefreshToken == "") {
		return perrors.WithCode(code.ErrInvalidArgument, "either access_token or refresh_token is required")
	}

	// 优先撤销 RefreshToken（更彻底）
	if req.RefreshToken != nil && *req.RefreshToken != "" {
		if err := s.tokenRefresher.RevokeRefreshToken(ctx, *req.RefreshToken); err != nil {
			return perrors.WithCode(code.ErrInvalidArgument, "failed to revoke refresh token: %v", err)
		}
		return nil
	}

	// 撤销 AccessToken
	if req.AccessToken != nil && *req.AccessToken != "" {
		if err := s.tokenIssuer.RevokeToken(ctx, *req.AccessToken); err != nil {
			return perrors.WithCode(code.ErrInvalidArgument, "failed to revoke access token: %v", err)
		}
		return nil
	}

	return nil
}

func (s *loginApplicationService) prepareAuthentication(req LoginRequest) (authentication.Scenario, authentication.AuthInput, error) {
	var scenario authentication.Scenario
	var authInput authentication.AuthInput

	switch req.AuthType {
	case AuthTypePassword:
		scenario = authentication.AuthPassword
		if err := s.validatePasswordFields(req); err != nil {
			return "", authInput, err
		}
		authInput = authentication.AuthInput{
			TenantID: req.TenantID,
			Username: *req.Username,
			Password: *req.Password,
		}

	case AuthTypePhoneOTP:
		scenario = authentication.AuthPhoneOTP
		if err := s.validatePhoneOTPFields(req); err != nil {
			return "", authInput, err
		}
		authInput = authentication.AuthInput{
			PhoneE164: *req.PhoneE164,
			OTP:       *req.OTPCode,
		}

	case AuthTypeWechat:
		scenario = authentication.AuthWxMinip
		if err := s.validateWechatFields(req); err != nil {
			return "", authInput, err
		}
		authInput = authentication.AuthInput{
			WxAppID:  *req.WechatAppID,
			WxJsCode: *req.WechatJSCode,
		}

	case AuthTypeWecom:
		scenario = authentication.AuthWecom
		if err := s.validateWecomFields(req); err != nil {
			return "", authInput, err
		}
		authInput = authentication.AuthInput{
			WecomCorpID: *req.WecomCorpID,
			WecomCode:   *req.WecomCode,
		}

	case AuthTypeJWTToken:
		scenario = authentication.AuthJWTToken
		if err := s.validateJWTTokenFields(req); err != nil {
			return "", authInput, err
		}
		authInput = authentication.AuthInput{
			AccessToken: *req.JWTToken,
		}

	default:
		return "", authInput, perrors.WithCode(code.ErrInvalidArgument, "unsupported auth type: %s", req.AuthType)
	}

	return scenario, authInput, nil
}

func (s *loginApplicationService) validatePasswordFields(req LoginRequest) error {
	if req.Username == nil || *req.Username == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "username is required for password authentication")
	}
	if req.Password == nil || *req.Password == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "password is required for password authentication")
	}
	return nil
}

func (s *loginApplicationService) validatePhoneOTPFields(req LoginRequest) error {
	if req.PhoneE164 == nil || *req.PhoneE164 == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "phone number is required for phone OTP authentication")
	}
	if req.OTPCode == nil || *req.OTPCode == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "OTP code is required for phone OTP authentication")
	}
	return nil
}

func (s *loginApplicationService) validateWechatFields(req LoginRequest) error {
	if req.WechatAppID == nil || *req.WechatAppID == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "wechat appid is required for wechat authentication")
	}
	if req.WechatJSCode == nil || *req.WechatJSCode == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "wechat jscode is required for wechat authentication")
	}
	return nil
}

func (s *loginApplicationService) validateWecomFields(req LoginRequest) error {
	if req.WecomCorpID == nil || *req.WecomCorpID == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "wecom corpid is required for wecom authentication")
	}
	if req.WecomCode == nil || *req.WecomCode == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "wecom code is required for wecom authentication")
	}
	return nil
}

func (s *loginApplicationService) validateJWTTokenFields(req LoginRequest) error {
	if req.JWTToken == nil || *req.JWTToken == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "jwt token is required for jwt token authentication")
	}
	return nil
}

func (s *loginApplicationService) convertAuthError(errCode authentication.ErrCode) error {
	switch errCode {
	case authentication.ErrInvalidCredential:
		return perrors.WithCode(code.ErrPasswordIncorrect, "invalid credentials")
	case authentication.ErrOTPMissingOrExpiry:
		return perrors.WithCode(code.ErrOTPInvalid, "OTP is invalid or expired")
	case authentication.ErrNoBinding:
		return perrors.WithCode(code.ErrNoBinding, "no account binding found")
	case authentication.ErrLocked:
		return perrors.WithCode(code.ErrCredentialLocked, "account is locked")
	case authentication.ErrDisabled:
		return perrors.WithCode(code.ErrCredentialDisabled, "account is disabled")
	case authentication.ErrIDPExchangeFailed:
		return perrors.WithCode(code.ErrIDPExchangeFailed, "failed to exchange with identity provider")
	case authentication.ErrStateMismatch:
		return perrors.WithCode(code.ErrStateMismatch, "state parameter mismatch")
	default:
		return perrors.WithCode(code.ErrAuthenticationFailed, "authentication failed")
	}
}
