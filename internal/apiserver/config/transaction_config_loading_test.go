package config

import (
	"strings"
	"testing"
	"time"

	options "github.com/FangcunMount/iam/v5/internal/apiserver/options"
	"github.com/FangcunMount/iam/v5/pkg/app"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestTransactionOwnerSurvivesFileConfiguration(t *testing.T) {
	for _, format := range []string{"yaml", "json"} {
		t.Run(format, func(t *testing.T) {
			reader := viper.New()
			reader.SetConfigType(format)
			input := `{"messaging":{"provider":"rocketmq","publish_enabled":true,"transactions":{"iam.mysql":{"store_id":"Fixture-Primary","writer_epoch":"m3-1","route_revision":"m3-1"}}}}`
			if format == "yaml" {
				input = `messaging:
  provider: rocketmq
  publish_enabled: true
  transactions:
    iam.mysql:
      store_id: Fixture-Primary
      writer_epoch: m3-1
      route_revision: m3-1
      limits:
        sends: 8
        branches: 16
        check_workers: 4
        check_queue: 32
        reconcile_batch: 100
        check_timeout: 2s
        rpc_timeout: 5s
        lease: 60s
        reconcile_interval: 5s
`
			}
			if err := reader.ReadConfig(strings.NewReader(input)); err != nil {
				t.Fatal(err)
			}
			flags := pflag.NewFlagSet("fixture", pflag.ContinueOnError)
			flags.Bool("messaging.publish_enabled", true, "fixture")
			if err := flags.Set("messaging.publish_enabled", "false"); err != nil {
				t.Fatal(err)
			}
			if err := reader.BindPFlags(flags); err != nil {
				t.Fatal(err)
			}
			opts := options.NewOptions()
			if err := app.DecodeOptions(reader, opts); err != nil {
				t.Fatal(err)
			}
			binding, ok := opts.Messaging.Transactions["iam.mysql"]
			if !ok || len(opts.Messaging.Transactions) != 1 || binding.StoreID != "Fixture-Primary" || binding.WriterEpoch != "m3-1" || binding.RouteRevision != "m3-1" {
				t.Fatalf("transaction owner identity or binding changed: %#v", opts.Messaging.Transactions)
			}
			if opts.Messaging.PublishEnabled {
				t.Fatal("CLI admission closure lost precedence")
			}
			limits, err := binding.ProtocolLimits("iam.mysql")
			if err != nil || limits.CheckTimeout != 2*time.Second || limits.Sends != 8 {
				t.Fatalf("transaction budget changed: %v", err)
			}
		})
	}
}

func TestTransactionConfigDecodeFailureIsNotIgnored(t *testing.T) {
	reader := viper.New()
	reader.SetConfigType("yaml")
	if err := reader.ReadConfig(strings.NewReader("messaging:\n  transactions: invalid\n")); err != nil {
		t.Fatal(err)
	}
	if err := app.DecodeOptions(reader, options.NewOptions()); err == nil {
		t.Fatal("malformed transaction configuration accepted")
	}
}
