package signin

import (
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestValidatePrincipalSessionAlignmentRejectsMismatchedUser(t *testing.T) {
	principal := &authentication.Principal{UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2)}
	sess := &sessiondomain.Session{UserID: meta.FromUint64(9), LoginIdentityID: meta.FromUint64(2), SessionID: "sid"}
	require.Error(t, validatePrincipalSessionAlignment(principal, sess))
}
