package testhelpers

import (
	"math/rand"
	"strings"
	"time"
)

// RetryOnDBLocked runs the provided operation and, if it fails with a
// sqlite "database is locked" (or similar transient) error, retries it with
// exponential backoff and jitter. This utility is intended for tests only to
// reduce noise from transient sqlite locking under heavy concurrency.
func RetryOnDBLocked(op func() error) error {
	const maxAttempts = 8
	baseDelay := 10 * time.Millisecond

	var lastErr error
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := op(); err != nil {
			lastErr = err
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "database is locked") || strings.Contains(msg, "database is busy") || strings.Contains(msg, "database table is locked") {
				sleep := baseDelay * (1 << attempt)
				if sleep > 500*time.Millisecond {
					sleep = 500 * time.Millisecond
				}
				jitter := time.Duration(r.Int63n(int64(sleep)))
				time.Sleep(sleep + jitter)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}
