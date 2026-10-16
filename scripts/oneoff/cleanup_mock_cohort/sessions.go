package main

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	keyspace "github.com/FangcunMount/component-base/pkg/redis/keyspace"
	sessionapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authn/session"
	sessiondomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/session"
	sessioncache "github.com/FangcunMount/iam/v5/internal/apiserver/infra/cache/redis"
	"github.com/redis/go-redis/v9"
)

type sessionEvidence struct {
	ID            string    `json:"session_id"`
	User          string    `json:"user_id"`
	LoginIdentity string    `json:"login_identity_id"`
	Expires       time.Time `json:"original_expires_at"`
	Status        string    `json:"status"`
}

func redisIdentity(ctx context.Context, c *redis.Client) (string, error) {
	info, err := c.Info(ctx, "server").Result()
	if err != nil {
		return "", err
	}
	return redisIdentityFromInfo(info, c.Options().DB)
}
func redisIdentityFromInfo(info string, db int) (string, error) {
	runID := ""
	for _, line := range strings.Split(info, "\n") {
		if strings.HasPrefix(line, "run_id:") {
			if runID != "" {
				return "", errors.New("multiple Redis server identities")
			}
			runID = strings.TrimSpace(strings.TrimPrefix(line, "run_id:"))
		}
	}
	if len(runID) != 40 || db < 0 {
		return "", errors.New("Redis run identity unavailable")
	}
	if _, err := strconv.ParseUint(runID[:16], 16, 64); err != nil {
		return "", errors.New("Redis run identity invalid")
	}
	return rawSHA([]byte(runID + "\x00" + strconv.Itoa(db))), nil
}

func inspectSessions(ctx context.Context, c *redis.Client, scope iamScope, revoke bool) ([]sessionEvidence, error) {
	store := sessioncache.NewSessionStore(c)
	revoker := sessionapp.NewRevoker(sessiondomain.NewRevoker(store))
	// Never bulk-revoke from an index: stale members may belong to retained users.

	// Index-only revocation misses a historical unindexed session. SCAN is bounded
	// and read-only outside the target set; only the existing Store may revoke.
	prefix := keyspace.New("session").Prefix("") + ":"
	users, logins := idSet(scope.Users), idSet(scope.LoginIdentities)
	var cursor uint64
	seen := map[string]bool{}
	out := []sessionEvidence{}
	scanned := 0
	for {
		keys, next, e := c.Scan(ctx, cursor, prefix+"*", 500).Result()
		if e != nil {
			return nil, e
		}
		scanned += len(keys)
		if scanned > 2_000_000 {
			return nil, errors.New("session scan bound exceeded")
		}
		for _, key := range keys {
			sid := strings.TrimPrefix(key, prefix)
			if seen[sid] {
				continue
			}
			seen[sid] = true
			s, e := store.Get(ctx, sid)
			if e != nil {
				return nil, e
			}
			if s == nil || !users[s.UserID.String()] && !logins[s.LoginIdentityID.String()] {
				continue
			}
			expires := s.ExpiresAt
			if revoke && s.IsActive() {
				if e = revoker.RevokeSession(ctx, sid, "mock_cohort_cleanup", "system:10001"); e != nil {
					return nil, e
				}
				s, e = store.Get(ctx, sid)
				if e != nil {
					return nil, e
				}
			}
			if revoke && s != nil && (s.IsActive() || !s.ExpiresAt.Equal(expires)) {
				return nil, errors.New("session revocation/TTL readback failed")
			}
			status := "expired_absent"
			if s != nil {
				status = string(s.Status)
			}
			out = append(out, sessionEvidence{sid, safeUser(s), safeLogin(s), expires, status})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if revoke {
		for _, x := range out {
			if x.Status == string(sessiondomain.StatusActive) {
				return nil, errors.New("active target session remains")
			}
		}
	}
	return out, nil
}
func safeUser(s *sessiondomain.Session) string {
	if s == nil {
		return ""
	}
	return s.UserID.String()
}
func safeLogin(s *sessiondomain.Session) string {
	if s == nil {
		return ""
	}
	return s.LoginIdentityID.String()
}
