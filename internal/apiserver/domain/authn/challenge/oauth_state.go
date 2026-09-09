package challenge

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"
)

const (
	OAuthPayloadKeyAppID       = "app_id"
	OAuthPayloadKeyRedirectURI = "redirect_uri"
	OAuthPayloadKeyNonce       = "nonce"
	OAuthPayloadKeyUserID      = "user_id"
	oauthStateRandomBytes      = 32
)

// OAuthState 是 OAuth state 挑战方式。
type OAuthState struct{}

// OAuthStateSpec OAuth state 挑战规格。
type OAuthStateSpec struct {
	// ---- 流程绑定 ----
	Scene       string // OAuth 挑战使用场景
	AppID       string // 发起 OAuth 流程的应用 ID
	RedirectURI string // 流程绑定的回调地址
	UserID      string // 可选的发起用户标识，供绑定流程恢复

	// ---- 流程校验值 ----
	Nonce string // 流程随机值，为空时生成
	State string // 回调校验值，为空时生成

	// ---- 生命周期参数 ----
	TTL time.Duration // 挑战有效时长
	Now time.Time     // 签发时间依据
}

// OAuthStateIssueResult OAuth state 签发结果。
type OAuthStateIssueResult struct {
	Challenge *AuthChallenge // 待保存的 OAuth 挑战
	State     string         // 交付 OAuth 流程的 state 值
	Nonce     string         // 随流程保存的随机值
	ExpiresAt time.Time      // 挑战失效时间
}

// Issue 创建 OAuth state 挑战实体。
func (OAuthState) Issue(spec OAuthStateSpec) (*OAuthStateIssueResult, error) {
	scene := strings.TrimSpace(spec.Scene)
	if scene == "" {
		return nil, ErrOAuthSceneRequired
	}
	appID := strings.TrimSpace(spec.AppID)
	redirectURI := strings.TrimSpace(spec.RedirectURI)
	if appID == "" {
		return nil, ErrAppIDRequired
	}
	if redirectURI == "" {
		return nil, ErrRedirectURIRequired
	}

	nonce := strings.TrimSpace(spec.Nonce)
	if nonce == "" {
		generated, err := randomOAuthToken()
		if err != nil {
			return nil, err
		}
		nonce = generated
	}
	state := strings.TrimSpace(spec.State)
	if state == "" {
		generated, err := randomOAuthToken()
		if err != nil {
			return nil, err
		}
		state = generated
	}

	ttl := spec.TTL
	if ttl <= 0 {
		ttl = DefaultOAuthStateTTL
	}
	now := normalizeVerificationTime(spec.Now)
	expiresAt := now.Add(ttl)

	payload := map[string]string{
		OAuthPayloadKeyAppID:       appID,
		OAuthPayloadKeyRedirectURI: redirectURI,
		OAuthPayloadKeyNonce:       nonce,
	}
	if userID := strings.TrimSpace(spec.UserID); userID != "" {
		payload[OAuthPayloadKeyUserID] = userID
	}

	challenge := &AuthChallenge{
		ID:         OAuthStateChallengeID(scene, state),
		Type:       TypeOAuthState,
		Scene:      scene,
		Target:     appID,
		SecretHash: OAuthStateSecretHash(state),
		Payload:    payload,
		ExpiresAt:  expiresAt,
		CreatedAt:  now,
	}
	return &OAuthStateIssueResult{
		Challenge: challenge,
		State:     state,
		Nonce:     nonce,
		ExpiresAt: expiresAt,
	}, nil
}

// IssueOAuthState 创建 OAuth state 挑战实体。
func IssueOAuthState(spec OAuthStateSpec) (*OAuthStateIssueResult, error) {
	return OAuthState{}.Issue(spec)
}

// OAuthStateContext OAuth state 消费后恢复的上下文。
type OAuthStateContext struct {
	AppID       string // 挑战绑定的应用 ID
	RedirectURI string // 挑战绑定的回调地址
	Nonce       string // 挑战保存的随机值
	UserID      string // 挑战保存的可选发起用户标识
}

// VerifyOAuthStateInput OAuth state 校验输入。
type VerifyOAuthStateInput struct {
	Scene string    // 需要匹配的使用场景
	State string    // 回调携带的 state 值
	Now   time.Time // 核验时间依据
}

// OAuthStateVerification OAuth state 校验输出。
type OAuthStateVerification struct {
	Context OAuthStateContext  // 验证成功后恢复的流程上下文
	Result  VerificationResult // 挑战验证及消费结果
}

// OAuthStateVerifier OAuth state 校验器。
type OAuthStateVerifier struct {
	Repo Repository
}

// NewOAuthStateVerifier 创建 OAuth state 校验器。
func NewOAuthStateVerifier(repo Repository) *OAuthStateVerifier {
	if repo == nil {
		return nil
	}
	return &OAuthStateVerifier{Repo: repo}
}

// VerifyAndConsume 校验并消费 OAuth state 挑战。
func (v *OAuthStateVerifier) VerifyAndConsume(ctx context.Context, input VerifyOAuthStateInput) (OAuthStateVerification, error) {
	if v == nil || v.Repo == nil {
		return OAuthStateVerification{Result: VerificationResult{Outcome: VerificationInfrastructureError}}, ErrRepositoryNotConfigured
	}
	scene := strings.TrimSpace(input.Scene)
	state := strings.TrimSpace(input.State)
	if state == "" {
		return OAuthStateVerification{Result: VerificationResult{Outcome: VerificationRejected}}, ErrOAuthStateRequired
	}
	now := normalizeVerificationTime(input.Now)

	challengeID := OAuthStateChallengeID(scene, state)
	challenge, err := v.Repo.Get(ctx, challengeID)
	if err != nil {
		return OAuthStateVerification{Result: VerificationResult{Outcome: VerificationInfrastructureError}}, err
	}
	if usabilityErr := oauthUsabilityError(AssessUsability(challenge, now, TypeOAuthState, scene)); usabilityErr != nil {
		return OAuthStateVerification{Result: VerificationResult{Outcome: VerificationRejected}}, usabilityErr
	}
	if !secretHashMatches(challenge.SecretHash, OAuthStateSecretHash(state)) {
		return OAuthStateVerification{Result: VerificationResult{Outcome: VerificationRejected}}, ErrOAuthStateMismatch
	}

	result, err := consumeOnce(ctx, v.Repo, challengeID, challenge.SecretHash)
	if err != nil {
		return OAuthStateVerification{Result: result}, err
	}
	if result.Outcome != VerificationSuccess {
		return OAuthStateVerification{Result: result}, ErrOAuthStateAlreadyUsed
	}
	return OAuthStateVerification{
		Context: oauthStateContextFromPayload(challenge.Payload),
		Result:  result,
	}, nil
}

func oauthUsabilityError(usability Usability) error {
	switch usability {
	case UsabilityOK:
		return nil
	case UsabilityNotFound:
		return ErrOAuthStateNotFound
	case UsabilityWrongType, UsabilityWrongScene:
		return ErrOAuthStateInvalid
	case UsabilityConsumed:
		return ErrOAuthStateAlreadyUsed
	case UsabilityExpired:
		return ErrOAuthStateExpired
	default:
		return ErrOAuthStateInvalid
	}
}

func oauthStateContextFromPayload(payload map[string]string) OAuthStateContext {
	if payload == nil {
		return OAuthStateContext{}
	}
	return OAuthStateContext{
		AppID:       payload[OAuthPayloadKeyAppID],
		RedirectURI: payload[OAuthPayloadKeyRedirectURI],
		Nonce:       payload[OAuthPayloadKeyNonce],
		UserID:      payload[OAuthPayloadKeyUserID],
	}
}

func randomOAuthToken() (string, error) {
	buf := make([]byte, oauthStateRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
