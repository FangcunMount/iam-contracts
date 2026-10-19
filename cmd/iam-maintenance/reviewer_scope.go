package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"time"

	crypto "github.com/FangcunMount/iam/v5/internal/apiserver/infra/crypto"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/accountprovision"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/FangcunMount/iam/v5/pkg/eventcatalog"
)

func runReviewerScope(args []string, output io.Writer) error {
	var protocolErr error
	if len(args) == 0 || (args[0] != "preflight" && args[0] != "apply") {
		return errors.New("reviewer-scope requires preflight or apply")
	}
	f := flag.NewFlagSet("reviewer-scope", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	inputPath := f.String("input", "", "original private provisioning input")
	reportPath := f.String("report", "", "new restricted report")
	orgRaw := f.String("org-id", "", "QS verified common company")
	fingerprint := f.String("fingerprint", "", "reviewed authorization facts")
	catalog := f.String("event-catalog", "configs/events.yaml", "event catalog")
	outboxMode := f.String("outbox-mode", "", "explicit standard or transaction protocol")
	transactionConfig := f.String("transaction-messaging-config", "", "explicit host-owned RocketMQ transaction options JSON")
	if err := f.Parse(args[1:]); err != nil || f.NArg() != 0 {
		return errors.New("invalid reviewer scope arguments")
	}
	org, err := meta.ParseID(*orgRaw)
	if err != nil || org <= 0 || org.String() != *orgRaw {
		return errors.New("canonical QS company required")
	}
	input, err := readAccountInput(*inputPath)
	if err != nil {
		return err
	}
	file, err := openScopeReport(*reportPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	db, err := roleDatabase("IAM_APISERVER_")
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return errors.New("IAM database unavailable")
	}
	defer func() { closeMaintenancePool(pool, protocolErr) }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	hasher := crypto.NewArgon2Hasher("")
	var report accountprovision.ReviewerScopeReport
	if args[0] == "preflight" {
		report, err = accountprovision.ReviewerScopePreflight(ctx, db, input, hasher, org)
	} else {
		cfg, e := eventcatalog.Load(*catalog)
		if e != nil {
			return errors.New("valid event catalog required")
		}
		err = withMaintenanceStager(ctx, db, eventcatalog.NewCatalog(cfg), *outboxMode, *transactionConfig, func(stager event.Stager) error {
			var businessErr error
			report, businessErr = accountprovision.ReviewerScopeApply(ctx, db, input, hasher, org, *fingerprint, stager)
			return businessErr
		})
		protocolErr = err
	}
	evidence := struct {
		Report accountprovision.ReviewerScopeReport `json:"report"`
		Error  string                               `json:"error,omitempty"`
	}{Report: report}
	if err != nil {
		evidence.Error = err.Error()
		if !maintenanceBusinessSucceeded(err) {
			evidence.Report.State = "unconfirmed"
		}
	}
	if e := json.NewEncoder(file).Encode(evidence); e != nil {
		return errors.New("cannot save private scope evidence; inspect before retry")
	}
	if e := file.Sync(); e != nil {
		return e
	}
	if err != nil {
		return errors.New("reviewer scope operation stopped; inspect private report")
	}
	return writeJSON(output, map[string]any{"state": report.State, "changes": len(report.Plan.Changes), "fingerprint": report.Fingerprint})
}
