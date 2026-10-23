package platform

import (
	"database/sql"
	"testing"
	"time"

	"github.com/FangcunMount/reliable-messaging/transaction"
)

func TestNativeNoticeResolutionValidationKeepsFirstIndexAndReceipt(t *testing.T) {
	b := transaction.Binding{Owner: "iam.mysql", StoreID: "primary", WriterEpoch: "epoch"}
	s, err := NewSQLNativeLimits(new(sql.DB), b, "m3-1")
	if err != nil {
		t.Fatal(err)
	}
	index, proof, receipt := [32]byte{1}, [32]byte{2}, [32]byte{3}
	key := nativeNoticeKey(b, "m3-1", index)
	row := sqlNativeNotice{key: key[:], owner: []byte(b.Owner), store: []byte(b.StoreID), epoch: []byte(b.WriterEpoch), revision: []byte("m3-1"), index: index[:], proof: proof[:], state: []byte("blocked"), schema: 1}
	if err = s.validate(row); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	row.state = []byte("resolved")
	row.resolvedAt = &now
	row.receipt = receipt[:]
	if err = s.validate(row); err != nil {
		t.Fatal("valid reviewed history rejected", err)
	}
	for _, change := range []struct {
		name   string
		mutate func(*sqlNativeNotice)
	}{
		{"missing-time", func(r *sqlNativeNotice) { r.resolvedAt = nil }},
		{"zero-time", func(r *sqlNativeNotice) { r.resolvedAt = &time.Time{} }},
		{"missing-receipt", func(r *sqlNativeNotice) { r.receipt = nil }},
		{"zero-receipt", func(r *sqlNativeNotice) { r.receipt = make([]byte, 32) }},
		{"short-receipt", func(r *sqlNativeNotice) { r.receipt = []byte{1} }},
		{"unknown-state", func(r *sqlNativeNotice) { r.state = []byte("ignored") }},
		{"blocked-with-resolution", func(r *sqlNativeNotice) { r.state = []byte("blocked") }},
		{"different-primary", func(r *sqlNativeNotice) { r.store = []byte("other") }},
		{"changed-index", func(r *sqlNativeNotice) { r.index = proof[:] }},
	} {
		t.Run(change.name, func(t *testing.T) {
			altered := row
			change.mutate(&altered)
			if s.validate(altered) == nil {
				t.Fatal("invalid resolved history accepted")
			}
		})
	}
}
