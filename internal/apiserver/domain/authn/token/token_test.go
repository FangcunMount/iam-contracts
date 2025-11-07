package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

func TestNewAccessAndRefreshTokensAndPair(t *testing.T) {
	userID := meta.FromUint64(1)
	acctID := meta.FromUint64(2)
	at := NewAccessToken("aid", "val", userID, acctID, time.Minute)
	rt := NewRefreshToken("rid", "rval", userID, acctID, time.Hour)

	assert.Equal(t, TokenTypeAccess, at.Type)
	assert.Equal(t, TokenTypeRefresh, rt.Type)
	pair := NewTokenPair(at, rt)
	assert.NotNil(t, pair)
	assert.Equal(t, "aid", pair.AccessToken.ID)
}

func TestTokenExpiryAndRemaining(t *testing.T) {
	userID := meta.FromUint64(3)
	acctID := meta.FromUint64(4)
	// expired token by negative duration
	texp := NewAccessToken("e", "v", userID, acctID, -time.Minute)
	assert.True(t, texp.IsExpired())
	assert.Equal(t, time.Duration(0), texp.RemainingDuration())

	// valid token
	tvalid := NewAccessToken("v", "val", userID, acctID, time.Minute)
	assert.False(t, tvalid.IsExpired())
	rd := tvalid.RemainingDuration()
	assert.Greater(t, rd, time.Duration(0))
}

func TestTokenClaimsExpiry(t *testing.T) {
	now := time.Now()
	uid := meta.FromUint64(5)
	aid := meta.FromUint64(6)
	claims := NewTokenClaims("tid", uid, aid, now.Add(-time.Minute), now.Add(-time.Second))
	assert.True(t, claims.IsExpired())

	uid2 := meta.FromUint64(7)
	aid2 := meta.FromUint64(8)
	claims2 := NewTokenClaims("tid2", uid2, aid2, now, now.Add(time.Minute))
	assert.False(t, claims2.IsExpired())
}
