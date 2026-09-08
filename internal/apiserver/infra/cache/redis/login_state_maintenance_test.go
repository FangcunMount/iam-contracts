package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestPurgeLoginStatePreviewAndApplyPreserveUnrelatedKeys(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()
	ctx := context.Background()
	keys := []string{sessionRedisKey("sid"), userSessionIndexRedisKey("1"), loginIdentitySessionIndexRedisKey("2"), refreshTokenRedisKey("secret"), consumedRefreshTokenRedisKey("secret"), revokedBearerTokenRedisKey("jti")}
	for _, key := range append(keys, "qs:cache:keep", challengeRedisKey("keep"), wechatAccessTokenRedisKey("keep")) {
		require.NoError(t, client.Set(ctx, key, "value", 0).Err())
	}
	preview, err := PurgeLoginState(ctx, client, 2, false)
	require.NoError(t, err)
	require.Len(t, preview.Families, 6)
	for _, counts := range preview.Families {
		require.EqualValues(t, 1, counts.Matched)
		require.Zero(t, counts.Deleted)
	}
	result, err := PurgeLoginState(ctx, client, 2, true)
	require.NoError(t, err)
	for _, counts := range result.Families {
		require.EqualValues(t, 1, counts.Deleted)
	}
	for _, key := range keys {
		require.False(t, mr.Exists(key))
	}
	require.True(t, mr.Exists("qs:cache:keep"))
	require.True(t, mr.Exists(challengeRedisKey("keep")))
	require.True(t, mr.Exists(wechatAccessTokenRedisKey("keep")))
	_, err = PurgeLoginState(ctx, client, 2, true)
	require.NoError(t, err)
}
