package sms

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginOTPSMSPayload_JSON(t *testing.T) {
	b, err := json.Marshal(LoginOTPSMSPayload{
		EventType: EventLoginOTPSMS,
		Scene:     "login",
		PhoneE164: "+8613800138000",
		Code:      "123456",
	})
	require.NoError(t, err)
	assert.Contains(t, string(b), "iam.login_otp_sms")
	assert.Contains(t, string(b), "+8613800138000")
}
