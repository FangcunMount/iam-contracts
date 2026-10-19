package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/operationscatalog"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/FangcunMount/iam/v5/pkg/eventcatalog"
	"io"
	"time"
)

func runOperationsCatalog(args []string, output io.Writer) error {
	var protocolErr error
	if len(args) == 0 || (args[0] != "preflight" && args[0] != "apply") {
		return errors.New("statistics-operations requires preflight or apply")
	}
	f := flag.NewFlagSet("statistics-operations", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	fp := f.String("fingerprint", "", "reviewed preflight fingerprint")
	actor := f.String("actor-id", "", "protected administrator ID")
	paused := f.Bool("writes-stopped", false, "authorization writes are paused")
	path := f.String("report", "", "new restricted report path")
	catalog := f.String("event-catalog", "configs/events.yaml", "event catalog")
	outboxMode := f.String("outbox-mode", "", "explicit standard or transaction protocol")
	transactionConfig := f.String("transaction-messaging-config", "", "explicit host-owned RocketMQ transaction options JSON")
	if e := f.Parse(args[1:]); e != nil {
		return e
	}
	if f.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	file, e := openScopeReport(*path)
	if e != nil {
		return e
	}
	defer func() { _ = file.Close() }()
	db, e := roleDatabase("IAM_APISERVER_")
	if e != nil {
		return e
	}
	pool, e := db.DB()
	if e != nil {
		return e
	}
	defer func() { closeMaintenancePool(pool, protocolErr) }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var report operationscatalog.Report
	if args[0] == "preflight" {
		report, e = operationscatalog.Preflight(ctx, db)
	} else {
		cfg, err := eventcatalog.Load(*catalog)
		if err != nil {
			return err
		}
		e = withMaintenanceStager(ctx, db, eventcatalog.NewCatalog(cfg), *outboxMode, *transactionConfig, func(stager event.Stager) error {
			var businessErr error
			report, businessErr = operationscatalog.Apply(ctx, db, stager, *fp, *actor, *paused)
			return businessErr
		})
		protocolErr = e
	}
	receipt := struct {
		Report operationscatalog.Report `json:"report"`
		Error  string                   `json:"error,omitempty"`
	}{Report: report}
	if e != nil {
		receipt.Error = e.Error()
	}
	if err := json.NewEncoder(file).Encode(receipt); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "statistics operations state=%s; report saved privately\n", report.State); err != nil {
		return err
	}
	if e != nil {
		return errors.New("operations configuration stopped; inspect private report")
	}
	return nil
}
