package options

import "errors"

var ErrTransactionStartupMigration = errors.New("transaction runtime requires explicit migration.enabled=false; install schema and bindings offline")

func (o *Options) ValidateTransactionStartupPolicy() error {
	if o != nil && len(o.Messaging.Transactions) > 0 && (o.MigrationOptions == nil || o.MigrationOptions.Enabled) {
		return ErrTransactionStartupMigration
	}
	return nil
}
