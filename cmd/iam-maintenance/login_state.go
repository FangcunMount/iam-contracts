package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"time"

	redisinfra "github.com/FangcunMount/iam/v5/internal/apiserver/infra/cache/redis"
	goredis "github.com/redis/go-redis/v9"
)

func runPurgeLoginState(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("purge-login-state", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	apply := flags.Bool("apply", false, "delete IAM login state")
	confirm := flags.String("confirm", "", "required for apply")
	batch := flags.Int64("batch-size", 500, "scan batch size")
	timeout := flags.Duration("timeout", 5*time.Minute, "operation timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *batch <= 0 || *timeout <= 0 {
		return errors.New("invalid purge-login-state arguments")
	}
	if (*apply && *confirm != "PURGE_IAM_LOGIN_STATE") || (!*apply && *confirm != "") {
		return errors.New("apply requires --confirm PURGE_IAM_LOGIN_STATE")
	}
	options, err := redisOptionsFromEnvironment()
	if err != nil {
		return err
	}
	client := goredis.NewClient(options)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := redisinfra.PurgeLoginState(ctx, client, *batch, *apply)
	if err != nil {
		return err
	}
	return writeJSON(output, struct {
		Mode string `json:"mode"`
		redisinfra.LoginStatePurgeResult
	}{map[bool]string{true: "apply", false: "dry-run"}[*apply], result})
}
