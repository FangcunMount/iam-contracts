package signin

import (
	"context"
	"errors"
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	tokenapp "github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/token"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
	"testing"
	"time"

	admissiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/admission"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	tokendomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestSignInCompletionRequiresAdmissionBeforeCreatingAuthenticationState(t *testing.T) {
	t.Parallel()

	principal := testPrincipal()
	creator := &recordingSessionCreator{session: testSession(principal)}
	minter := &recordingTokenSetMinter{}
	saver := &recordingRefreshTokenSaver{}
	policy := admissionPolicyStub{decision: admissiondomain.Deny(
		admissiondomain.Subject{UserID: principal.UserID, LoginIdentityID: principal.LoginIdentityID},
		admissiondomain.ReasonUserBlocked,
	)}
	establisher := newCompletionForTest(completionTestDependencies{
		AdmissionPolicy: policy, SessionCreator: creator, SessionRevoker: &recordingSessionRevoker{}, TokenSetMinter: minter, RefreshTokenSaver: saver,
	})

	result, err := establisher.completeAuthentication(context.Background(), principal, sessiondomain.TokenContext{})

	require.Nil(t, result)
	require.Equal(t, code.ErrUserBlocked, perrors.ParseCoder(err).Code())
	require.False(t, creator.called)
	require.False(t, minter.called)
	require.False(t, saver.called)
}

func TestSignInCompletionDoesNotCreateAuthenticationStateWhenAdmissionCannotBeEvaluated(t *testing.T) {
	t.Parallel()

	principal := testPrincipal()
	creator := &recordingSessionCreator{session: testSession(principal)}
	establisher := newCompletionForTest(completionTestDependencies{
		AdmissionPolicy: admissionPolicyStub{err: errors.New("status unavailable")},
		SessionCreator:  creator,
	})

	result, err := establisher.completeAuthentication(context.Background(), principal, sessiondomain.TokenContext{})

	require.Nil(t, result)
	var evaluation *admissiondomain.EvaluationError
	require.ErrorAs(t, err, &evaluation)
	require.False(t, creator.called)
}

func TestSignInCompletionCreatesResultAndPersistsInitialRefreshToken(t *testing.T) {
	t.Parallel()

	principal := testPrincipal()
	sess := testSession(principal)
	refresh := tokendomain.NewRefreshToken(
		"refresh-id", "refresh-value", sess.SessionID,
		principal.UserID, principal.LoginIdentityID, principal.TenantID,
		nil, nil, time.Hour,
	)
	set := tokendomain.NewUserTokenSet(
		tokendomain.NewAccessToken(
			"access-id", "access-value", sess.SessionID,
			principal.UserID, principal.LoginIdentityID, principal.TenantID,
			time.Minute,
		),
		refresh,
	)
	creator := &recordingSessionCreator{session: sess}
	minter := &recordingTokenSetMinter{set: set}
	saver := &recordingRefreshTokenSaver{}
	establisher := newCompletionForTest(completionTestDependencies{
		AdmissionPolicy: admissionPolicyStub{decision: admissiondomain.Admit(
			admissiondomain.Subject{UserID: principal.UserID, LoginIdentityID: principal.LoginIdentityID},
		)},
		SessionCreator: creator, SessionRevoker: &recordingSessionRevoker{}, TokenSetMinter: minter, RefreshTokenSaver: saver,
	})

	tokenContext := sessiondomain.TokenContext{TenantDomain: "fangcun", OrgID: meta.FromUint64(42)}
	result, err := establisher.completeAuthentication(context.Background(), principal, tokenContext)

	require.NoError(t, err)
	require.Equal(t, tokenContext, creator.tokenContext)
	require.Equal(t, principal.UserID, result.UserID)
	require.Equal(t, set.AccessToken.Value, result.TokenPair.AccessToken.Value)
	require.Same(t, sess, minter.session)
	require.Same(t, refresh, saver.token)
}

type admissionPolicyStub struct {
	decision admissiondomain.Decision
	err      error
}

func (s admissionPolicyStub) Evaluate(context.Context, admissiondomain.Subject) (admissiondomain.Decision, error) {
	return s.decision, s.err
}

type recordingSessionCreator struct {
	tokenContext sessiondomain.TokenContext
	session      *sessiondomain.Session
	called       bool
}

func (s *recordingSessionCreator) Create(_ context.Context, _ *authentication.Principal, tokenContext sessiondomain.TokenContext) (*sessiondomain.Session, error) {
	s.called = true
	s.tokenContext = tokenContext.Clone()
	return s.session, nil
}

type recordingTokenSetMinter struct {
	err     error
	set     *tokendomain.UserTokenSet
	session *sessiondomain.Session
	called  bool
}

func (m *recordingTokenSetMinter) MintTokenSet(_ context.Context, session *sessiondomain.Session) (*tokendomain.UserTokenSet, error) {
	m.called = true
	m.session = session
	return m.set, m.err
}

type recordingRefreshTokenSaver struct {
	err    error
	token  *tokendomain.RefreshToken
	called bool
}

func (s *recordingRefreshTokenSaver) SaveRefreshToken(_ context.Context, token *tokendomain.RefreshToken) error {
	s.called = true
	s.token = token
	return s.err
}

func testPrincipal() *authentication.Principal {
	return &authentication.Principal{
		UserID:          meta.FromUint64(1),
		LoginIdentityID: meta.FromUint64(2),
		TenantID:        meta.FromUint64(3),
	}
}

func testSession(principal *authentication.Principal) *sessiondomain.Session {
	return sessiondomain.NewWithContexts(
		"session-id", principal.UserID, principal.LoginIdentityID, principal.TenantID,
		principal.AuthContext, sessiondomain.TokenContext{}, time.Now().Add(time.Hour),
	)
}

type recordingSessionRevoker struct {
	sessionID, reason string
	err               error
	contextErr        error
	deadline          bool
}

func (r *recordingSessionRevoker) Revoke(ctx context.Context, sid, reason, _ string) error {
	r.sessionID, r.reason = sid, reason
	r.contextErr = ctx.Err()
	_, r.deadline = ctx.Deadline()
	return r.err
}

func TestSignInCompletionCompensatesFailedEstablishmentEvenAfterRequestCancellation(t *testing.T) {
	for _, stage := range []string{"mint", "save", "incomplete"} {
		t.Run(stage, func(t *testing.T) {
			principal := testPrincipal()
			sess := testSession(principal)
			failure := errors.New("injected grant failure")
			minter := &recordingTokenSetMinter{set: &tokendomain.UserTokenSet{
				AccessToken: &tokendomain.AccessToken{}, RefreshToken: &tokendomain.RefreshToken{},
			}}
			saver := &recordingRefreshTokenSaver{}
			switch stage {
			case "mint":
				minter.err = failure
			case "save":
				saver.err = failure
			case "incomplete":
				minter.set = nil
			}
			for _, cleanupFails := range []bool{false, true} {
				revoker := &recordingSessionRevoker{}
				if cleanupFails {
					revoker.err = errors.New("injected cleanup failure")
				}
				establisher := newCompletionForTest(completionTestDependencies{
					AdmissionPolicy: admissionPolicyStub{decision: admissiondomain.Admit(admissiondomain.Subject{})},
					SessionCreator:  &recordingSessionCreator{session: sess}, SessionRevoker: revoker,
					TokenSetMinter: minter, RefreshTokenSaver: saver,
				})
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				result, err := establisher.completeAuthentication(ctx, principal, sessiondomain.TokenContext{})
				require.Nil(t, result)
				require.Error(t, err)
				if stage != "incomplete" {
					require.ErrorIs(t, err, failure)
				}
				if cleanupFails {
					require.ErrorIs(t, err, revoker.err)
				}
				require.Equal(t, sess.SessionID, revoker.sessionID)
				require.Equal(t, "authentication_grant_failed", revoker.reason)
				require.NoError(t, revoker.contextErr)
				require.True(t, revoker.deadline)
			}
		})
	}
}

func TestSignInCompletionRequiresCompensationBeforeCreatingSession(t *testing.T) {
	creator := &recordingSessionCreator{}
	establisher := newCompletionForTest(completionTestDependencies{
		AdmissionPolicy: admissionPolicyStub{decision: admissiondomain.Admit(admissiondomain.Subject{})},
		SessionCreator:  creator, TokenSetMinter: &recordingTokenSetMinter{}, RefreshTokenSaver: &recordingRefreshTokenSaver{},
	})
	_, err := establisher.completeAuthentication(context.Background(), testPrincipal(), sessiondomain.TokenContext{})
	require.Error(t, err)
	require.False(t, creator.called)
}

func TestSignInCompletionRejectsMismatchedSessionBeforeMintingAndCompensates(t *testing.T) {
	principal := testPrincipal()
	sess := testSession(principal)
	sess.LoginIdentityID = meta.FromUint64(99)
	minter := &recordingTokenSetMinter{}
	saver := &recordingRefreshTokenSaver{}
	revoker := &recordingSessionRevoker{}
	establisher := newCompletionForTest(completionTestDependencies{
		AdmissionPolicy: admissionPolicyStub{decision: admissiondomain.Admit(admissiondomain.Subject{})},
		SessionCreator:  &recordingSessionCreator{session: sess}, SessionRevoker: revoker,
		TokenSetMinter: minter, RefreshTokenSaver: saver,
	})
	result, err := establisher.completeAuthentication(context.Background(), principal, sessiondomain.TokenContext{})
	require.Error(t, err)
	require.Nil(t, result)
	require.False(t, minter.called)
	require.False(t, saver.called)
	require.Equal(t, sess.SessionID, revoker.sessionID)
}

type completionTestDependencies struct {
	AdmissionPolicy   admissiondomain.Policy
	SessionCreator    sessiondomain.Creator
	SessionRevoker    SessionRevoker
	TokenSetMinter    tokendomain.TokenSetMinter
	RefreshTokenSaver tokenapp.RefreshTokenSaver
}

func newCompletionForTest(d completionTestDependencies) *SignIn {
	var issuer tokenapp.InitialTokenIssuer
	if d.TokenSetMinter != nil && d.RefreshTokenSaver != nil {
		issuer = tokenapp.NewInitialTokenIssuer(d.TokenSetMinter, d.RefreshTokenSaver)
	}
	return New(Dependencies{AdmissionPolicy: d.AdmissionPolicy, SessionCreator: d.SessionCreator, SessionRevoker: d.SessionRevoker, TokenIssuer: issuer})
}
