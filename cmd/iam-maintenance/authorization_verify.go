package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	authzruntime "github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/runtime"
)

func runAuthorizationVerify(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("authorization-verify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return err
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	dataset, err := authzruntime.NewMySQLSource(db).Load(ctx)
	if err != nil {
		return err
	}
	snapshot, err := authzruntime.BuildSnapshot(dataset, time.Now().UTC())
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "authorization snapshot verified: version=%d roles=%d assignments=%d grants=%d\n", snapshot.Version(), len(dataset.Roles), len(dataset.Assignments), len(dataset.Grants))
	return err
}
