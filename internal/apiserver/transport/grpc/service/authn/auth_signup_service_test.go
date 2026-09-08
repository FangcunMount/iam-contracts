package authn

import (
	"testing"

	authnv3 "github.com/FangcunMount/iam/v5/api/grpc/iam/authn/v3"
	signupApp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authn/signup"
	"github.com/stretchr/testify/require"
)

func TestWechatMiniProgramSignupRequestFromGRPCUsesOnlyCodeProof(t *testing.T) {
	t.Parallel()

	mapped, err := wechatMiniProgramSignupRequestFromGRPC(&authnv3.SignUpWithWechatMiniProgramRequest{
		Name:   "alice",
		AppId:  " wx-app ",
		JsCode: " js-code ",
	})
	require.NoError(t, err)

	identity, ok := mapped.LoginIdentity.(signupApp.WechatMiniLoginIdentityInput)
	require.True(t, ok)
	require.Equal(t, "wx-app", *identity.AppID)
	require.Equal(t, "js-code", *identity.JsCode)
	require.Nil(t, identity.OpenID)
	require.Nil(t, identity.UnionID)
}
