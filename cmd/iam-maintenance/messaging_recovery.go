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

func runMessagingRecovery(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("m3-messaging-recovery", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	store := flags.String("store-id", "", "desired frozen primary binding")
	epoch := flags.String("writer-epoch", "", "frozen writer epoch")
	export := flags.String("export-protection", "", "new private original protection")
	protection := flags.String("protection", "", "original private protection file")
	reviewed := flags.String("reviewed-sha256", "", "independent protection digest")
	restore := flags.Bool("restore", false, "restore only seven initially absent objects and original journal row")
	excluded := flags.Bool("writers-excluded", false, "operator attestation; not writer exclusion proof")
	prestart := flags.Bool("pre-start-restore-attested", false, "only from original coordinator pre-start-restore; attestation not proof")
	timeout := flags.Duration("timeout", 2*time.Minute, "budget, at most five minutes")
	if flags.Parse(args) != nil || flags.NArg() != 0 || *timeout <= 0 || *timeout > 5*time.Minute {
		return errors.New("invalid M3 protection arguments")
	}
	if *restore && (!*excluded || !*prestart || *protection == "" || *reviewed == "" || *export != "" || *store != "" || *epoch != "") {
		return errors.New("M3 restore requires original reviewed protection and both external attestations")
	}
	if !*restore && (*protection != "" || *reviewed != "" || *excluded || *prestart || *export == "") {
		return errors.New("M3 protection requires a new private export and desired binding")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	open := func() (*sql.DB, error) {
		db, err := roleDatabaseOptions("IAM_APISERVER_", true)
		if err != nil {
			return nil, errors.New("M3 protection database unavailable")
		}
		pool, err := db.DB()
		if err != nil {
			return nil, errors.New("M3 protection database unavailable")
		}
		pool.SetMaxOpenConns(4)
		return pool, nil
	}
	if *restore {
		info, err := os.Lstat(*protection)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 131072 {
			return errors.New("M3 original private protection unavailable")
		}
		raw, err := os.ReadFile(*protection)
		if err != nil {
			return errors.New("M3 original private protection unavailable")
		}
		p, err := messagingcutover.DecodeProtection(raw, *reviewed)
		if err != nil {
			return err
		}
		if err = messagingcutover.RestoreProtection(ctx, p, open); err != nil {
			return err
		}
		return writeJSON(output, map[string]any{"status": "original_scope_restore_returned", "cutover_authorized": false, "business_tables_restored": false})
	}
	db, err := open()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	p, err := messagingcutover.CaptureProtection(ctx, db, *store, *epoch)
	if err != nil {
		return err
	}
	raw, err := messagingcutover.EncodeProtection(p)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(*export, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("M3 protection output must be new")
	}
	_, writeErr := file.Write(raw)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("M3 original protection publication incomplete")
	}
	sha := sha256.Sum256(raw)
	return writeJSON(output, map[string]any{"status": "original_scope_protection_exported", "sha256": hex.EncodeToString(sha[:]), "cutover_authorized": false})
}
