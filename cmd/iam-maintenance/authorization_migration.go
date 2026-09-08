package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/FangcunMount/iam/v5/internal/pkg/migration"
)

// runAuthorizationMigration 用于单一授权空间切换的两个明确阶段；不提供降级路径。
func runAuthorizationMigration(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("authorization-migrate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	target := flags.Uint("to", 0, "31 establishes protection; 32 removes the old partition")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || (*target != 31 && *target != 32) {
		return errors.New("authorization-migrate requires --to 31 or --to 32")
	}
	db, err := authzConvergeDatabaseFromEnvironment()
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()
	database := firstEnvironment("MYSQL_DATABASE", "MYSQL_DBNAME")
	version, applied, err := migration.NewMigrator(pool, &migration.Config{Enabled: true, Database: database}).RunTo(uint(*target))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "authorization migration completed: version=%d applied=%t\n", version, applied)
	return err
}
