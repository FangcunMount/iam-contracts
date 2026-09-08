package authentication

import (
	credDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/credential"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDecisionRejectsContradictoryFacts(t *testing.T) {
	principal := &Principal{UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2)}
	for _, tc := range []struct {
		name     string
		decision AuthDecision
	}{
		{"success without principal", AuthDecision{OK: true}},
		{"rejection with principal", AuthDecision{Principal: principal}},
		{"success with rejection code", AuthDecision{OK: true, Principal: principal, Code: 123}},
		{"duplicate identity on success", AuthDecision{OK: true, Principal: principal, RejectedLoginIdentityID: meta.FromUint64(3)}},
		{"failure updates success state", AuthDecision{CredentialUpdate: &CredentialUpdate{CredentialID: meta.FromUint64(4), Effect: CredentialEffectRecordSuccess}}},
		{"success updates failure state", AuthDecision{OK: true, Principal: principal, CredentialUpdate: &CredentialUpdate{CredentialID: meta.FromUint64(4), Effect: CredentialEffectRecordFailure}}},
		{"rotation without material", AuthDecision{OK: true, Principal: principal, CredentialUpdate: &CredentialUpdate{CredentialID: meta.FromUint64(4), Effect: CredentialEffectRecordSuccess, Rotation: &credDomain.MaterialRotation{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Error(t, tc.decision.Validate()) })
	}
	decision := AuthDecision{OK: true, Principal: principal}
	require.NoError(t, decision.Validate())
	require.Equal(t, principal.LoginIdentityID, decision.LoginIdentityID())
}
