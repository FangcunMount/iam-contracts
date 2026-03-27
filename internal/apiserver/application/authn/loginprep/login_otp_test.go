package loginprep

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_randomNumericOTP(t *testing.T) {
	s, err := randomNumericOTP(6)
	require.NoError(t, err)
	assert.Len(t, s, 6)
	for _, c := range s {
		assert.True(t, c >= '0' && c <= '9')
	}

	_, err = randomNumericOTP(0)
	require.Error(t, err)
}

func TestPhoneOTPDeps_effective(t *testing.T) {
	var d *PhoneOTPDeps
	assert.Equal(t, 60, int(d.effectiveCooldown().Seconds()))
	assert.Equal(t, 5*60, int(d.effectiveTTL().Seconds()))
	assert.Equal(t, 6, d.effectiveCodeLen())

	d = &PhoneOTPDeps{Cooldown: 0, TTL: 0, CodeLen: 0}
	assert.Equal(t, 6, d.effectiveCodeLen())
}
