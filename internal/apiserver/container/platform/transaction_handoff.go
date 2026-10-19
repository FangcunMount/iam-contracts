package platform

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
)

// A bounded read prevents a new transaction runtime from silently abandoning
// old publication responsibilities. It does not prove exclusion of old writers
// or authorize a production cutover; the frozen handoff still supplies those.
func checkTransactionHandoff(ctx context.Context, db *gorm.DB, budget time.Duration) error {
	check, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	for _, table := range []struct{ name, column string }{{"domain_event_outbox", "status"}, {"rm_outbox", "state"}} {
		var id struct{ ID uint64 }
		result := db.WithContext(check).Table(table.name).Select("id").Where(table.column+" IS NULL OR "+table.column+" <> ?", "published").Limit(1).Find(&id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 0 {
			return errors.New("old authorization publication responsibilities must be drained before transaction startup")
		}
	}
	return nil
}
