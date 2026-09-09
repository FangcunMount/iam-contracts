package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/eventoutbox"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/rolemodel"
	"github.com/FangcunMount/iam/v5/pkg/eventcatalog"
	drivermysql "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func runRoleModelMigration(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("role-model-migrate requires preflight, apply, verify, or rollback")
	}
	mode := args[0]
	if mode != "preflight" && mode != "apply" && mode != "verify" && mode != "rollback" {
		return errors.New("invalid role migration operation")
	}
	f := flag.NewFlagSet("role-model-migrate", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	fingerprint := f.String("fingerprint", "", "reviewed source fingerprint")
	stopped := f.Bool("writes-stopped", false, "authorization and relevant identity writers are stopped")
	catalogPath := f.String("event-catalog", "configs/events.yaml", "durable event catalog")
	timeout := f.Duration("timeout", time.Minute, "operation timeout")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 || *timeout <= 0 {
		return errors.New("invalid role migration arguments")
	}
	if (mode == "apply" || mode == "rollback") && (!*stopped || len(*fingerprint) != 64) {
		return errors.New("role migration mutation requires --writes-stopped and --fingerprint")
	}
	iam, err := roleDatabase("")
	if err != nil {
		return err
	}
	pool, err := iam.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	var qs *gorm.DB
	if mode == "preflight" || mode == "apply" {
		qs, err = roleDatabase("QS_")
		if err != nil {
			return err
		}
		qpool, err := qs.DB()
		if err != nil {
			return err
		}
		defer qpool.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	switch mode {
	case "preflight":
		p, err := rolemodel.Preflight(ctx, iam, qs)
		if err != nil {
			return err
		}
		if err = writeJSON(output, p); err != nil {
			return err
		}
		return p.Validate()
	case "verify":
		r, err := rolemodel.Verify(ctx, iam)
		if err != nil {
			return err
		}
		return writeJSON(output, r)
	default:
		cfg, err := eventcatalog.Load(*catalogPath)
		if err != nil {
			return err
		}
		stager := eventoutbox.NewStore(iam, eventcatalog.NewCatalog(cfg))
		var receipt *rolemodel.Receipt
		if mode == "apply" {
			receipt, err = rolemodel.Apply(ctx, iam, qs, stager, *fingerprint, *stopped)
		} else {
			receipt, err = rolemodel.Rollback(ctx, iam, stager, *fingerprint, *stopped)
		}
		if err != nil {
			return err
		}
		return writeJSON(output, receipt)
	}
}

// Credentials remain in process environment. Driver Config escapes special
// characters correctly and connection failures never print a DSN.
func roleDatabase(prefix string) (*gorm.DB, error) {
	get := func(key string) string { return strings.TrimSpace(os.Getenv(prefix + "MYSQL_" + key)) }
	host := get("HOST")
	port := get("PORT")
	if port == "" {
		port = "3306"
	}
	if h, p, err := net.SplitHostPort(host); err == nil {
		host, port = h, p
	}
	user := get("USERNAME")
	if user == "" {
		user = get("USER")
	}
	database := get("DATABASE")
	if database == "" {
		database = get("DBNAME")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 || host == "" || user == "" || database == "" {
		return nil, errors.New(prefix + "MYSQL connection environment is invalid")
	}
	cfg := drivermysql.NewConfig()
	cfg.User = user
	cfg.Passwd = os.Getenv(prefix + "MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, port)
	cfg.DBName = database
	cfg.ParseTime = true
	cfg.Timeout = 8 * time.Second
	cfg.ReadTimeout = time.Minute
	cfg.WriteTimeout = time.Minute
	db, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, errors.New(prefix + "MYSQL connection failed")
	}
	return db, nil
}
