package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestOTPVerifierVerifyAndConsume(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	verifier := NewOTPVerifier(client)
	ctx := context.Background()

	if err := verifier.Put(ctx, "+8613800138000", "login", "123456", time.Minute); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	if ok := verifier.VerifyAndConsume(ctx, "+8613800138000", "login", "123456"); !ok {
		t.Fatalf("expected first VerifyAndConsume() to succeed")
	}
	if ok := verifier.VerifyAndConsume(ctx, "+8613800138000", "login", "123456"); ok {
		t.Fatalf("expected second VerifyAndConsume() to fail")
	}
}
