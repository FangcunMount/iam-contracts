package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"strings"

	"github.com/golang-migrate/migrate/v4/source"
)

var ErrInstalledHead = errors.New("installed schema does not match this runtime; run approved offline migration")

// VerifyInstalledHead borrows the business pool and only reads its existing journal.
// It deliberately does not construct the migrate MySQL driver, which can create metadata.
func VerifyInstalledHead(ctx context.Context, db *sql.DB) error {
	if ctx == nil || db == nil {
		return fmt.Errorf("%w: context and original pool required", ErrInstalledHead)
	}
	if _, ok := ctx.Deadline(); !ok {
		return fmt.Errorf("%w: bounded read required", ErrInstalledHead)
	}
	head, err := embeddedHead(migrations)
	if err != nil {
		return fmt.Errorf("%w: embedded migration head: %v", ErrInstalledHead, err)
	}
	rows, err := db.QueryContext(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 2")
	if err != nil {
		return fmt.Errorf("%w: journal read: %v", ErrInstalledHead, err)
	}
	defer func() { _ = rows.Close() }()
	count := 0
	for rows.Next() {
		var version int64
		var dirty bool
		if err := rows.Scan(&version, &dirty); err != nil {
			return fmt.Errorf("%w: invalid journal row: %v", ErrInstalledHead, err)
		}
		count++
		if count != 1 || version != int64(head) || dirty {
			return fmt.Errorf("%w: require one clean row at %d", ErrInstalledHead, head)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: journal read: %v", ErrInstalledHead, err)
	}
	if count != 1 {
		return fmt.Errorf("%w: require one clean row at %d", ErrInstalledHead, head)
	}
	return nil
}

func embeddedHead(fsys fs.FS) (uint, error) {
	entries, err := fs.ReadDir(fsys, "migrations")
	if err != nil {
		return 0, err
	}
	seen := make(map[uint]bool)
	var head uint
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		m, err := source.DefaultParse(entry.Name())
		if err != nil {
			return 0, err
		}
		if m.Version == 0 || uint64(m.Version) > math.MaxInt64 || seen[m.Version] {
			return 0, errors.New("invalid or duplicate migration version")
		}
		seen[m.Version] = true
		if m.Version > head {
			head = m.Version
		}
	}
	if head == 0 {
		return 0, errors.New("no up migration")
	}
	return head, nil
}
