package options

import (
	"errors"
	"testing"
)

func TestTransactionStartupRequiresOfflineMigrationWithoutChangingLegacyMode(t *testing.T) {
	for _, tc := range []struct {
		name                                string
		transaction, enabled, missing, want bool
	}{
		{"legacy-auto", false, true, false, false}, {"legacy-disabled", false, false, false, false},
		{"transaction-auto", true, true, false, true}, {"transaction-disabled", true, false, false, false},
		{"transaction-missing", true, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := NewOptions()
			o.MigrationOptions.Enabled = tc.enabled
			if tc.missing {
				o.MigrationOptions = nil
			}
			if tc.transaction {
				o.Messaging.Provider = "rocketmq"
				o.Messaging.Transactions = map[string]TransactionOptions{"iam.mysql": {StoreID: "primary", WriterEpoch: "m3-1", RouteRevision: "m3-1"}}
			}
			found := false
			for _, err := range o.Validate() {
				found = found || errors.Is(err, ErrTransactionStartupMigration)
			}
			if found != tc.want {
				t.Fatalf("offline policy error=%v want=%v", found, tc.want)
			}
		})
	}
}
