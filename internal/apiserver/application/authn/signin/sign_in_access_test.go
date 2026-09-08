package signin

import (
	"context"
	"github.com/FangcunMount/iam/v4/internal/apiserver/testhelpers"
	"testing"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	credentialapp "github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/credential"
	authnexternal "github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/externalidentity"
	"github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/signin/method"
	tokenapp "github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/token"
	idpresolver "github.com/FangcunMount/iam/v4/internal/apiserver/application/idp/externalidentity"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	idpidentity "github.com/FangcunMount/iam/v4/internal/apiserver/domain/idp/externalidentity"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
)

func TestSignInPreservesInitialTokenIssuerErrorCodes(t *testing.T) {
	tests := []struct {
		name      string
		issueCode int
	}{
		{name: "issued"},
		{name: "blocked user", issueCode: code.ErrUserBlocked},
		{name: "disabled identity", issueCode: code.ErrLoginIdentityDisabled},
		{name: "inactive user", issueCode: code.ErrUserInactive},
		{name: "admission evaluation failure", issueCode: code.ErrInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principal := &authentication.Principal{
				UserID:          meta.FromUint64(1),
				LoginIdentityID: meta.FromUint64(2),
			}
			initialTokenIssuer := &initialTokenIssuerStub{errCode: tt.issueCode}
			strategy := signInStrategyStub{decision: authentication.AuthDecision{OK: true, Principal: principal}}
			usecase := New(Dependencies{
				TokenIssuer:     initialTokenIssuer,
				AdmissionPolicy: testhelpers.AuthnFlow{}, SessionCreator: testhelpers.AuthnFlow{}, SessionRevoker: testhelpers.AuthnFlow{},
				MethodRegistry: signInMethodRegistryStub{},
				ProofFactory:   signInProofFactoryStub{},
				Authenticator:  authentication.NewAuthenticator(strategy),
			})

			result, err := usecase.Execute(context.Background(), method.LoginRequest{})
			if tt.issueCode == 0 {
				if err != nil || result == nil {
					t.Fatalf("Execute() result = %#v, err = %v", result, err)
				}
			} else {
				if got := perrors.ParseCoder(err).Code(); got != tt.issueCode {
					t.Fatalf("Execute() error code = %d, want %d, err = %v", got, tt.issueCode, err)
				}
				if result != nil {
					t.Fatalf("Execute() result = %#v, want nil", result)
				}
			}
			if !initialTokenIssuer.called {
				t.Fatal("IssueInitialTokens() was not called")
			}
		})
	}
}

func TestSignInRecordsCredentialBeforeIssuingInitialTokens(t *testing.T) {
	t.Parallel()

	principal := &authentication.Principal{
		UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2)}
	order := make([]string, 0, 2)
	initialTokenIssuer := &initialTokenIssuerStub{order: &order}
	usecase := New(Dependencies{
		TokenIssuer:     initialTokenIssuer,
		AdmissionPolicy: testhelpers.AuthnFlow{}, SessionCreator: testhelpers.AuthnFlow{}, SessionRevoker: testhelpers.AuthnFlow{},
		MethodRegistry: signInMethodRegistryStub{},
		ProofFactory:   signInProofFactoryStub{},
		Authenticator: authentication.NewAuthenticator(signInStrategyStub{decision: authentication.AuthDecision{
			OK: true, Principal: principal,

			CredentialUpdate: &authentication.CredentialUpdate{CredentialID: meta.FromUint64(4)},
		}}),
		CredentialRecorder: credentialRecorderStub{order: &order},
	})

	result, err := usecase.Execute(context.Background(), method.LoginRequest{})

	if err != nil || result == nil {
		t.Fatalf("Execute() result = %#v, err = %v", result, err)
	}
	if len(order) != 2 || order[0] != "record" || order[1] != "issue" {
		t.Fatalf("call order = %v, want [record issue]", order)
	}
}

func TestSignInProviderExchangeFailureKeepsPublicContract(t *testing.T) {
	resolutionErr := &idpresolver.ResolutionError{
		Kind:     idpresolver.ErrorProviderExchange,
		Provider: idpidentity.ProviderWechatMinip,
		Realm:    "mini-app",
	}
	proofErr := authnexternal.MapLoginProofError(context.Background(), resolutionErr, "wechat_minip")
	usecase := New(Dependencies{
		MethodRegistry: signInMethodRegistryStub{},
		ProofFactory:   signInProofFactoryErrorStub{err: proofErr},
	})

	credential, err := usecase.buildCredential(context.Background(), method.LoginRequest{})
	if credential != nil {
		t.Fatalf("buildCredential() credential = %#v, want nil", credential)
	}
	coder := perrors.ParseCoder(err)
	if got := coder.Code(); got != code.ErrInternalServerError {
		t.Fatalf("buildCredential() error code = %d, want %d, err = %v", got, code.ErrInternalServerError, err)
	}
	if got := coder.String(); got != "Internal server error" {
		t.Fatalf("buildCredential() public message = %q, want %q", got, "Internal server error")
	}
}

type signInCredentialStub struct{}

func (signInCredentialStub) CredentialKind() authentication.CredentialKind {
	return authentication.CredentialKindPassword
}

type signInMethodRegistryStub struct{}

func (signInMethodRegistryStub) Select(context.Context, method.LoginRequest) (method.LoginMethodSelection, error) {
	return method.LoginMethodSelection{CredentialKind: method.CredentialKindPassword}, nil
}

type signInProofFactoryStub struct{}

func (signInProofFactoryStub) Build(context.Context, method.LoginMethodSelection) (authentication.IdentityProof, error) {
	return signInCredentialStub{}, nil
}

type signInProofFactoryErrorStub struct {
	err error
}

func (s signInProofFactoryErrorStub) Build(context.Context, method.LoginMethodSelection) (authentication.IdentityProof, error) {
	return nil, s.err
}

type signInStrategyStub struct {
	decision authentication.AuthDecision
}

func (signInStrategyStub) Kind() authentication.CredentialKind {
	return authentication.CredentialKindPassword
}

func (s signInStrategyStub) Authenticate(context.Context, authentication.IdentityProof) (authentication.AuthDecision, error) {
	return s.decision, nil
}

type credentialRecorderStub struct {
	order *[]string
}

var _ credentialapp.Recorder = credentialRecorderStub{}

func (s credentialRecorderStub) Record(context.Context, authentication.AuthDecision) error {
	*s.order = append(*s.order, "record")
	return nil
}

type initialTokenIssuerStub struct {
	called  bool
	errCode int
	order   *[]string
}

func (s *initialTokenIssuerStub) IssueInitialTokens(_ context.Context, principal *sessiondomain.Session) (*tokenapp.TokenPair, error) {
	s.called = true
	if s.order != nil {
		*s.order = append(*s.order, "issue")
	}
	if s.errCode != 0 {
		return nil, perrors.WithCode(s.errCode, "authentication grant denied")
	}
	return tokenapp.NewTokenPair(
		tokenapp.NewAccessToken("a", "access", "session-id", principal.UserID, principal.LoginIdentityID, principal.TenantID, time.Now(), time.Now().Add(time.Minute)),
		tokenapp.NewRefreshToken("r", "refresh", "session-id", principal.UserID, principal.LoginIdentityID, principal.TenantID, time.Now(), time.Now().Add(time.Hour)),
	), nil
}

// 租户仅从登录请求进入会话；身份核验结果不承载或确认租户归属。
func TestSignInCarriesRequestedTenantOutsidePrincipal(t *testing.T) {
	principal := &authentication.Principal{UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2)}
	for _, tenantID := range []meta.ID{meta.ZeroID, meta.FromUint64(42)} {
		usecase := New(Dependencies{
			TokenIssuer:     &initialTokenIssuerStub{},
			AdmissionPolicy: testhelpers.AuthnFlow{}, SessionCreator: testhelpers.AuthnFlow{}, SessionRevoker: testhelpers.AuthnFlow{},
			MethodRegistry: signInMethodRegistryStub{}, ProofFactory: signInProofFactoryStub{},
			Authenticator: authentication.NewAuthenticator(signInStrategyStub{decision: authentication.AuthDecision{OK: true, Principal: principal}}),
		})
		result, err := usecase.Execute(context.Background(), method.LoginRequest{TenantID: tenantID})
		if err != nil {
			t.Fatal(err)
		}
		if result.TenantID != tenantID || result.TokenPair.AccessToken.TenantID != tenantID || result.TokenPair.RefreshToken.TenantID != tenantID {
			t.Fatal("requested tenant was not preserved through session issuance")
		}
	}
}

func TestSignInRejectsContradictoryDecisionBeforeRecording(t *testing.T) {
	order := []string{}
	issuer := &initialTokenIssuerStub{}
	usecase := New(Dependencies{
		TokenIssuer: issuer, AdmissionPolicy: testhelpers.AuthnFlow{}, SessionCreator: testhelpers.AuthnFlow{}, SessionRevoker: testhelpers.AuthnFlow{},
		MethodRegistry: signInMethodRegistryStub{}, ProofFactory: signInProofFactoryStub{},
		Authenticator: authentication.NewAuthenticator(signInStrategyStub{decision: authentication.AuthDecision{
			OK: true, Principal: &authentication.Principal{UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2)},
			CredentialUpdate: &authentication.CredentialUpdate{CredentialID: meta.FromUint64(3), Effect: authentication.CredentialEffectRecordFailure},
		}}), CredentialRecorder: credentialRecorderStub{order: &order},
	})
	result, err := usecase.Execute(context.Background(), method.LoginRequest{})
	if err == nil || result != nil || len(order) != 0 || issuer.called {
		t.Fatal("invalid decision must stop before credential recording and token issuance")
	}
}
