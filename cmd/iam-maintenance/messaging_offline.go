package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/messagingcutover"
)

func runMessagingOffline(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("m3-messaging", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	action := flags.String("action", "schema", "schema or select")
	store := flags.String("store-id", "", "frozen primary binding")
	epoch := flags.String("writer-epoch", "", "frozen protocol writer epoch")
	export := flags.String("export-plan", "", "new private preview file")
	planPath := flags.String("plan", "", "independently reviewed private plan")
	reviewed := flags.String("reviewed-sha256", "", "independent original file digest")
	apply := flags.Bool("apply", false, "apply only through reviewed pre-start coordinator")
	excluded := flags.Bool("writers-excluded", false, "operator attestation, not machine proof")
	timeout := flags.Duration("timeout", 2*time.Minute, "operation budget, at most five minutes")
	if flags.Parse(args) != nil || flags.NArg() != 0 || *timeout <= 0 || *timeout > 5*time.Minute {
		return errors.New("invalid M3 offline arguments")
	}
	if *apply && (!*excluded || *planPath == "" || *reviewed == "" || *export != "" || *store != "" || *epoch != "" || *action != "schema") {
		return errors.New("M3 apply requires only original reviewed plan and writer exclusion attestation")
	}
	if !*apply && (*planPath != "" || *reviewed != "" || *excluded || *export == "") {
		return errors.New("M3 preview requires a new private plan path and desired binding")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	open := func() (*sql.DB, error) {
		db, err := roleDatabaseOptions("IAM_APISERVER_", true)
		if err != nil {
			return nil, errors.New("M3 database connection unavailable")
		}
		pool, err := db.DB()
		if err != nil {
			return nil, errors.New("M3 database connection unavailable")
		}
		pool.SetMaxOpenConns(4)
		return pool, nil
	}
	if *apply {
		info, err := os.Lstat(*planPath)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 65536 {
			return errors.New("M3 private original plan unavailable")
		}
		raw, err := os.ReadFile(*planPath)
		if err != nil {
			return errors.New("M3 private original plan unavailable")
		}
		p, err := messagingcutover.Decode(raw, *reviewed)
		if err != nil {
			return err
		}
		if err = messagingcutover.Apply(ctx, p, open); err != nil {
			return err
		}
		return writeJSON(output, map[string]any{"status": "offline_step_returned", "action": p.Action, "cutover_authorized": false})
	}
	db, err := open()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	snapshot, err := messagingcutover.Read(ctx, db)
	if err != nil {
		return err
	}
	p, err := messagingcutover.Build(*action, *store, *epoch, snapshot)
	if err != nil {
		return err
	}
	raw, err := messagingcutover.Encode(p)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(*export, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("M3 preview output must be new private file")
	}
	_, writeErr := file.Write(raw)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("M3 private preview publication incomplete")
	}
	sha := sha256.Sum256(raw)
	return writeJSON(output, map[string]any{"status": "read_only_plan_exported", "sha256": hex.EncodeToString(sha[:]), "cutover_authorized": false})
}
