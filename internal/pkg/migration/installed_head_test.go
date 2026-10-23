package migration

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
)

func TestEmbeddedSchemaHead(t *testing.T) {
	head, err := embeddedHead(migrations)
	if err != nil || head != 45 {
		t.Fatalf("expected frozen schema head 45, got %d: %v", head, err)
	}
	for _, tc := range []struct {
		name  string
		files fstest.MapFS
		head  uint
		bad   bool
	}{
		{"sorted-numerically", fstest.MapFS{"migrations/2_first.up.sql": {}, "migrations/10_last.up.sql": {}, "migrations/11_only.down.sql": {}}, 10, false},
		{"duplicate", fstest.MapFS{"migrations/2_a.up.sql": {}, "migrations/02_b.up.sql": {}}, 0, true},
		{"zero", fstest.MapFS{"migrations/0_a.up.sql": {}}, 0, true},
		{"overflow", fstest.MapFS{"migrations/9223372036854775808_a.up.sql": {}}, 0, true},
		{"malformed", fstest.MapFS{"migrations/bad.up.sql": {}}, 0, true},
		{"empty", fstest.MapFS{"migrations/1_a.down.sql": {}}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := embeddedHead(tc.files)
			if (err != nil) != tc.bad || (!tc.bad && got != tc.head) {
				t.Fatalf("head=%d err=%v", got, err)
			}
		})
	}
}

func TestInstalledHeadRequiresBorrowedPoolAndBoundedRead(t *testing.T) {
	if err := VerifyInstalledHead(context.Background(), nil); !errors.Is(err, ErrInstalledHead) {
		t.Fatal(err)
	}
}
