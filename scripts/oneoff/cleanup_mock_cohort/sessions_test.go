package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	keyspace "github.com/FangcunMount/component-base/pkg/redis/keyspace"
	sessiondomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/session"
	sessioncache "github.com/FangcunMount/iam/v5/internal/apiserver/infra/cache/redis"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSessionRevocationIncludesUnindexedAndPreservesTTL(t *testing.T) {
	m := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: m.Addr()})
	defer func() { _ = c.Close() }()
	store := sessioncache.NewSessionStore(c)
	ctx := context.Background()
	expires := time.Now().Add(time.Hour).UTC()
	for _, s := range []*sessiondomain.Session{{SessionID: "indexed", UserID: meta.ID(101), LoginIdentityID: meta.ID(301), Status: sessiondomain.StatusActive, CreatedAt: time.Now(), ExpiresAt: expires}, {SessionID: "shared-kept", UserID: meta.ID(999), LoginIdentityID: meta.ID(999), Status: sessiondomain.StatusActive, CreatedAt: time.Now(), ExpiresAt: expires}} {
		if e := store.Save(ctx, s); e != nil {
			t.Fatal(e)
		}
	}
	// Deliberately absent from both ZSETs; the real legacy decoder and Store
	// revocation must still find and revoke it without rewriting another user.
	legacy := map[string]any{"SessionID": "unindexed", "UserID": uint64(101), "LoginIdentityID": uint64(301), "Status": "active", "CreatedAt": time.Now(), "ExpiresAt": expires}
	b, _ := json.Marshal(legacy)
	if e := c.Set(ctx, keyspace.New("session").Prefix("unindexed"), b, time.Hour).Err(); e != nil {
		t.Fatal(e)
	}
	scope := iamScope{[]string{"101"}, []string{}, []string{"301"}}
	// A stale target index pointing at a retained user's session must never be
	// used as ownership evidence for bulk revocation.
	if e := c.ZAdd(ctx, keyspace.New("user_session_index").Prefix("101"), redis.Z{Score: float64(expires.Unix()), Member: "shared-kept"}).Err(); e != nil {
		t.Fatal(e)
	}
	if e := c.ZAdd(ctx, keyspace.New("login_identity_session_index").Prefix("301"), redis.Z{Score: float64(expires.Unix()), Member: "shared-kept"}).Err(); e != nil {
		t.Fatal(e)
	}
	before, e := inspectSessions(ctx, c, scope, false)
	if e != nil || len(before) != 2 {
		t.Fatalf("preflight %d %v", len(before), e)
	}
	if _, e = inspectSessions(ctx, c, scope, true); e != nil {
		t.Fatal(e)
	}
	for _, sid := range []string{"indexed", "unindexed"} {
		s, e := store.Get(ctx, sid)
		if e != nil || s == nil || s.IsActive() || s.Status != sessiondomain.StatusRevoked || !s.ExpiresAt.Equal(expires) {
			t.Fatalf("revocation/TTL mismatch %s %#v %v", sid, s, e)
		}
		ttl := m.TTL(keyspace.New("session").Prefix(sid))
		if ttl <= 0 || ttl > time.Hour {
			t.Fatal("revoked record retention lost")
		}
	}
	kept, e := store.Get(ctx, "shared-kept")
	if e != nil || !kept.IsActive() {
		t.Fatal("shared session changed")
	}
	if members, e := c.ZRange(ctx, keyspace.New("user_session_index").Prefix("101"), 0, -1).Result(); e != nil || len(members) != 1 || members[0] != "shared-kept" {
		t.Fatal("target session membership not removed or foreign member changed")
	}
	// A DB restore has no route to Save/Extend these sessions. Reinspection is
	// purely read-only and cannot reactivate the retained revoked payloads.
	if _, e = inspectSessions(ctx, c, scope, false); e != nil {
		t.Fatal(e)
	}
	for _, sid := range []string{"indexed", "unindexed"} {
		s, _ := store.Get(ctx, sid)
		if s.IsActive() {
			t.Fatal("readback revived session")
		}
	}
}
func TestRedisIdentityBindsRunAndLogicalDatabase(t *testing.T) {
	a, e := redisIdentityFromInfo("# Server\r\nrun_id:0123456789012345678901234567890123456789\r\n", 0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := redisIdentityFromInfo("run_id:0123456789012345678901234567890123456789\n", 1)
	if e != nil || a == b {
		t.Fatal("Redis DB binding absent")
	}
	for _, info := range []string{"", "run_id:bad", "run_id:0123456789012345678901234567890123456789\nrun_id:0123456789012345678901234567890123456789"} {
		if _, e = redisIdentityFromInfo(info, 0); e == nil {
			t.Fatal("unknown Redis identity accepted")
		}
	}
}
