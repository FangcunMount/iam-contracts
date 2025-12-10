package login

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	tokenDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	idpPort "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/logger"
)

type loginApplicationService struct {
	tokenIssuer      tokenDomain.Issuer
	tokenRefresher   tokenDomain.Refresher
	authenticater    *authentication.Authenticater
	wechatAppQuerier idpPort.Repository
	secretVault      idpPort.SecretVault
}

var _ LoginApplicationService = (*loginApplicationService)(nil)

func NewLoginApplicationService(
	tokenIssuer tokenDomain.Issuer,
	tokenRefresher tokenDomain.Refresher,
	authenticater *authentication.Authenticater,
	wechatAppQuerier idpPort.Repository,
	secretVault idpPort.SecretVault,
) LoginApplicationService {
	return &loginApplicationService{
		tokenIssuer:      tokenIssuer,
		tokenRefresher:   tokenRefresher,
		authenticater:    authenticater,
		wechatAppQuerier: wechatAppQuerier,
		secretVault:      secretVault,
	}
}

func (s *loginApplicationService) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	l := logger.L(ctx)

	scenario, authInput, err := s.prepareAuthentication(ctx, req)
	if err != nil {
		l.Warnw("认证准备失败",
			"action", logger.ActionLogin,
			"error", err.Error(),
		)
		return nil, err
	}

	l.Debugw("开始认证流程",
		"action", logger.ActionLogin,
		"scenario", string(scenario),
		"tenant_id", authInput.TenantID,
	)

	decision, err := s.authenticater.Authenticate(ctx, scenario, authInput)
	if err != nil {
		l.Errorw("认证过程异常",
			"action", logger.ActionLogin,
			"scenario", string(scenario),
			"error", err.Error(),
		)
		return nil, err
	}

	if !decision.OK {
		l.Warnw("认证失败",
			"action", logger.ActionLogin,
			"scenario", string(scenario),
			"err_code", string(decision.ErrCode),
			"credential_id", decision.CredentialID.String(),
			"result", logger.ResultFailed,
		)
		return nil, s.convertAuthError(decision.ErrCode)
	}

	l.Infow("认证成功，开始颁发令牌",
		"action", logger.ActionLogin,
		"scenario", string(scenario),
		"user_id", decision.Principal.UserID.String(),
		"account_id", decision.Principal.AccountID.String(),
		"should_rotate", decision.ShouldRotate,
	)

	tokenPair, err := s.tokenIssuer.IssueToken(ctx, decision.Principal)
	if err != nil {
		l.Errorw("令牌颁发失败",
			"action", logger.ActionLogin,
			"user_id", decision.Principal.UserID.String(),
			"error", err.Error(),
			"result", logger.ResultFailed,
		)
		return nil, perrors.WithCode(code.ErrInvalidArgument, "failed to issue token: %v", err)
	}

	l.Infow("登录完成",
		"action", logger.ActionLogin,
		"user_id", decision.Principal.UserID.String(),
		"account_id", decision.Principal.AccountID.String(),
		"tenant_id", decision.Principal.TenantID,
		"result", logger.ResultSuccess,
	)

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
	l := logger.L(ctx)

	// 至少需要提供一个令牌
	if (req.AccessToken == nil || *req.AccessToken == "") &&
		(req.RefreshToken == nil || *req.RefreshToken == "") {
		l.Warnw("登出请求缺少令牌",
			"action", logger.ActionLogout,
			"result", logger.ResultFailed,
		)
		return perrors.WithCode(code.ErrInvalidArgument, "either access_token or refresh_token is required")
	}

	// 优先撤销 RefreshToken（更彻底）
	if req.RefreshToken != nil && *req.RefreshToken != "" {
		if err := s.tokenRefresher.RevokeRefreshToken(ctx, *req.RefreshToken); err != nil {
			l.Errorw("撤销刷新令牌失败",
				"action", logger.ActionLogout,
				"error", err.Error(),
				"result", logger.ResultFailed,
			)
			return perrors.WithCode(code.ErrInvalidArgument, "failed to revoke refresh token: %v", err)
		}
		l.Infow("刷新令牌已撤销",
			"action", logger.ActionLogout,
			"result", logger.ResultSuccess,
		)
		return nil
	}

	// 撤销 AccessToken
	if req.AccessToken != nil && *req.AccessToken != "" {
		if err := s.tokenIssuer.RevokeToken(ctx, *req.AccessToken); err != nil {
			l.Errorw("撤销访问令牌失败",
				"action", logger.ActionLogout,
				"error", err.Error(),
				"result", logger.ResultFailed,
			)
			return perrors.WithCode(code.ErrInvalidArgument, "failed to revoke access token: %v", err)
		}
		l.Infow("访问令牌已撤销",
			"action", logger.ActionLogout,
			"result", logger.ResultSuccess,
		)
		return nil
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

func (s *loginApplicationService) prepareAuthentication(ctx context.Context, req LoginRequest) (authentication.Scenario, authentication.AuthInput, error) {
	// 构建统一的 AuthInput，根据请求中有哪些字段就填充哪些字段
	input := authentication.AuthInput{
		TenantID: req.TenantID,
	}

	// 根据存在的字段来推断认证场景
	var scenario authentication.Scenario

	// 密码认证
	if req.Username != nil && req.Password != nil {
		scenario = authentication.AuthPassword
		input.Username = *req.Username
		input.Password = *req.Password
	}

	// 手机号OTP认证
	if req.PhoneE164 != nil && req.OTPCode != nil {
		scenario = authentication.AuthPhoneOTP
		input.PhoneE164 = *req.PhoneE164
		input.OTP = *req.OTPCode
	}

	// 微信小程序认证
	if req.WechatAppID != nil && req.WechatJSCode != nil {
		scenario = authentication.AuthWxMinip
		input.WxAppID = *req.WechatAppID
		input.WxJsCode = *req.WechatJSCode

		// 查询微信应用配置获取 AppSecret
		if s.wechatAppQuerier != nil && s.secretVault != nil {
			wechatApp, err := s.wechatAppQuerier.GetByAppID(ctx, *req.WechatAppID)
			if err != nil {
				return "", authentication.AuthInput{}, perrors.WithCode(code.ErrInvalidArgument, "failed to query wechat app: %v", err)
			}
			if wechatApp == nil {
				return "", authentication.AuthInput{}, perrors.WithCode(code.ErrInvalidArgument, "wechat app not found: %s", *req.WechatAppID)
			}
			if !wechatApp.IsEnabled() {
				return "", authentication.AuthInput{}, perrors.WithCode(code.ErrInvalidArgument, "wechat app is disabled: %s", *req.WechatAppID)
			}
			if wechatApp.Cred == nil || wechatApp.Cred.Auth == nil {
				return "", authentication.AuthInput{}, perrors.WithCode(code.ErrInvalidArgument, "wechat app credentials not found")
			}

			appSecretPlain, err := s.secretVault.Decrypt(ctx, wechatApp.Cred.Auth.AppSecretCipher)
			if err != nil {
				return "", authentication.AuthInput{}, perrors.WithCode(code.ErrInvalidArgument, "failed to decrypt app secret: %v", err)
			}
			input.WxAppSecret = string(appSecretPlain)
		} else {
			return "", authentication.AuthInput{}, perrors.WithCode(code.ErrInvalidArgument, "wechat app configuration service not available")
		}
	}

	// 企业微信认证
	if req.WecomCorpID != nil && req.WecomCode != nil {
		scenario = authentication.AuthWecom
		input.WecomCorpID = *req.WecomCorpID
		input.WecomCode = *req.WecomCode
		// TODO: 查询企业微信应用配置获取 AgentID 和 CorpSecret
	}

	// JWT令牌认证
	if req.JWTToken != nil {
		scenario = authentication.AuthJWTToken
		input.AccessToken = *req.JWTToken
	}

	// 检查是否确定了认证场景
	if scenario == "" {
		return "", authentication.AuthInput{}, perrors.WithCode(code.ErrInvalidArgument, "no valid authentication credentials provided")
	}

	return scenario, input, nil
}
