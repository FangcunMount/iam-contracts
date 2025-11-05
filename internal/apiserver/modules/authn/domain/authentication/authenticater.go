package authentication

import "context"

// Authenticater 认证器
type Authenticater struct {
}

// Authenticate 认证
func (a *Authenticater) Authenticate(ctx context.Context, kind Scenario, in AuthInput) (AuthDecision, error) {
	switch kind {
	case AuthPassword:
		// Implement password authentication logic here
		return AuthDecision{}, nil
	default:
		return AuthDecision{}, nil
	}
}
