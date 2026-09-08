package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance"
)

func runTenantRetirement(args []string, output io.Writer) error {
	if len(args) == 0 || (args[0] != "preflight" && args[0] != "prepare") {
		return errors.New("tenant-retirement requires preflight or prepare")
	}
	flags := flag.NewFlagSet("tenant-retirement", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	fingerprint := flags.String("fingerprint", "", "required preflight fingerprint for prepare")
	timeout := flags.Duration("timeout", 5*time.Minute, "operation timeout")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *timeout <= 0 {
		return errors.New("invalid tenant-retirement arguments")
	}
	db, err := authzConvergeDatabaseFromEnvironment()
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	var report *maintenance.TenantRetirementReport
	if args[0] == "prepare" {
		unlock, lockErr := acquireAuthzV3ConvergeLock(ctx, pool, *timeout)
		if lockErr != nil {
			return lockErr
		}
		defer unlock()
		report, err = maintenance.PrepareTenantRetirement(ctx, db, *fingerprint)
	} else {
		report, err = maintenance.AnalyzeTenantRetirement(ctx, db)
	}
	if report != nil {
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(report); encodeErr != nil {
			return encodeErr
		}
	}
	if err != nil {
		return err
	}
	if report == nil || !report.Ready {
		return errors.New("授权数据预检未通过，未执行迁移")
	}
	return nil
}
