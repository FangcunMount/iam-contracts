package process

import (
	"errors"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/config"
	"github.com/FangcunMount/iam/v5/internal/apiserver/options"
)

func TestTransactionStartupRejectsDDLBeforeDatabaseResources(t *testing.T) {
	o := options.NewOptions()
	o.Messaging.Provider = "rocketmq"
	o.Messaging.Transactions = map[string]options.TransactionOptions{"iam.mysql": {}}
	// No database/Redis handles can be constructed by this pure rejection case.
	o.MySQLOptions = nil
	o.RedisOptions = nil
	dm := NewDatabaseManager(&config.Config{Options: o})
	if err := dm.Initialize(); !errors.Is(err, options.ErrTransactionStartupMigration) {
		t.Fatalf("startup policy not enforced before resources: %v", err)
	}
	if err := dm.runMigrations(); !errors.Is(err, options.ErrTransactionStartupMigration) {
		t.Fatalf("direct migration path bypassed policy: %v", err)
	}
}
