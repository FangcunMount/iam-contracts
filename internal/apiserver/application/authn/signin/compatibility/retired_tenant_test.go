package compatibility

import (
	"encoding/json"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authn/signin/method"
	"github.com/stretchr/testify/require"
)

func TestPasswordLoginIgnoresRetiredNumericTenant(t *testing.T) {
	for _, payload := range []string{
		`{"username":"alice","password":"secret"}`,
		`{"username":"alice","password":"secret","tenant_id":42}`,
		`{"username":"alice","password":"secret","tenant_id":999}`,
	} {
		request, err := BuildExplicitWireLoginRequest("password", json.RawMessage(payload))
		require.NoError(t, err)
		require.Equal(t, method.PasswordPayload{Username: "alice", Password: "secret"}, request.Payload)
		wire, err := json.Marshal(PasswordWirePayload{Username: "alice", Password: "secret"})
		require.NoError(t, err)
		require.NotContains(t, string(wire), "tenant_id")
	}
}
