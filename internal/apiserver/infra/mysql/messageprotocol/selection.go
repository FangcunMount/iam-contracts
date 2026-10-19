// Package messageprotocol reads the host's explicit publication-protocol gate.
// The gate is separate from SDK evidence, which remains readable across a
// reviewed compatibility rollback. Constructors never install or select it.
package messageprotocol

import (
	"context"
	"errors"
	"math"

	"github.com/FangcunMount/reliable-messaging/transaction"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const Table = "iam_authz_message_protocol"
const SchemaMigration = 43

var ErrSelection = errors.New("authorization publication protocol selection is missing or conflicts with the configured writer")
var errTableMissing = errors.New("authorization publication protocol table not installed")

type selection struct {
	Owner                               string
	ProtocolType                        string
	StoreID, WriterEpoch, RouteRevision string
	Generation                          uint64
	SchemaVersion                       int
}

func read(ctx context.Context, db *gorm.DB, lock bool) (selection, error) {
	var row selection
	if ctx == nil || db == nil {
		return row, ErrSelection
	}
	query := db.WithContext(ctx).Table(Table).Where("owner = ?", "iam.mysql")
	if lock {
		query = query.Clauses(clause.Locking{Strength: "SHARE"})
	}
	err := query.Take(&row).Error
	var mysqlErr *driver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
		return row, errTableMissing
	}
	if err != nil {
		return row, errors.Join(ErrSelection, err)
	}
	if row.Owner != "iam.mysql" || row.SchemaVersion != 1 || row.Generation < 1 || row.Generation > math.MaxInt64 {
		return row, ErrSelection
	}
	return row, nil
}

// RequireOutbox permits the historical pre-43 schema, but never a missing gate
// in a database whose migration head already declares the new schema installed.
func RequireOutbox(ctx context.Context, db *gorm.DB, lock bool) error {
	row, err := read(ctx, db, lock)
	if errors.Is(err, errTableMissing) {
		var versions []struct {
			Version uint64
			Dirty   bool
		}
		currentErr := db.WithContext(ctx).Table("schema_migrations").Limit(2).Find(&versions).Error
		if currentErr != nil || len(versions) != 1 || versions[0].Dirty || versions[0].Version >= SchemaMigration {
			return ErrSelection
		}
		return nil
	}
	if err != nil {
		return err
	}
	if row.ProtocolType != "outbox" {
		return ErrSelection
	}
	return nil
}
func RequireTransaction(ctx context.Context, db *gorm.DB, binding transaction.Binding, revision string, lock bool) error {
	row, err := read(ctx, db, lock)
	if err != nil {
		return errors.Join(ErrSelection, err)
	}
	if binding.Owner != "iam.mysql" || row.ProtocolType != "transaction" || row.StoreID != binding.StoreID || row.WriterEpoch != binding.WriterEpoch || revision != "m3-1" || row.RouteRevision != revision {
		return ErrSelection
	}
	return nil
}
