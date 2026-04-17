package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestAccessTokenCacheTryLockRefresh(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	cache := NewAccessTokenCache(client)
	ctx := context.Background()

	ok, unlock, err := cache.TryLockRefresh(ctx, "app-1", 10*time.Second)
	if err != nil {
		t.Fatalf("TryLockRefresh() error = %v", err)
	}
	if !ok || unlock == nil {
		t.Fatalf("expected first lock acquisition to succeed")
	}

	ok, unlock2, err := cache.TryLockRefresh(ctx, "app-1", 10*time.Second)
	if err != nil {
		t.Fatalf("TryLockRefresh() second error = %v", err)
	}
	if ok || unlock2 != nil {
		t.Fatalf("expected second lock acquisition to fail while held")
	}

	unlock()

	ok, unlock3, err := cache.TryLockRefresh(ctx, "app-1", 10*time.Second)
	if err != nil {
		t.Fatalf("TryLockRefresh() third error = %v", err)
	}
	if !ok || unlock3 == nil {
		t.Fatalf("expected lock acquisition after unlock to succeed")
	}
}
