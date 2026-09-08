package testhelpers

import (
	"context"
	admission "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/admission"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	session "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"time"
)

// AuthnFlow provides in-memory admission and session collaborators for application tests.
type AuthnFlow struct{}

func (AuthnFlow) Evaluate(_ context.Context, s admission.Subject) (admission.Decision, error) {
	return admission.Admit(s), nil
}
func (AuthnFlow) Create(_ context.Context, p *authentication.Principal, c session.TokenContext) (*session.Session, error) {
	return session.NewWithContexts("session-id", p.UserID, p.LoginIdentityID, p.TenantID, p.AuthContext, c, time.Now().Add(time.Hour)), nil
}
func (AuthnFlow) Revoke(context.Context, string, string, string) error { return nil }
