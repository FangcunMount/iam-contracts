package dbops_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletedCutoverNeverRepeatsPersonnelMigration(t *testing.T) {
	for _, state := range []string{"applied_unchanged", "applied_drifted", "rolled_back"} {
		t.Run(state, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "maintenance")
			calls := filepath.Join(dir, "calls")
			script := `#!/bin/sh
echo "$2" >> "$CALLS"
case "$2" in
status) printf '{"state":"%s","next_action":"none","fingerprint":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}\n' "$STATE" ;;
archive-inheritance) printf '{}\n' ;;
*) exit 99 ;;
esac
`
			if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			catalog := filepath.Join(dir, "events.yaml")
			if err := os.WriteFile(catalog, []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "role-model-migrate.sh")
			cmd.Env = append(os.Environ(), "STATE="+state, "CALLS="+calls,
				"IAM_ROLE_MODEL_MODE=cutover", "IAM_ROLE_MODEL_CONFIRM=ROLE_MODEL_MIGRATE", "IAM_ROLE_MODEL_WRITES_STOPPED=true",
				"IAM_ROLE_MODEL_BIN="+bin, "IAM_ROLE_MODEL_CATALOG="+catalog, "IAM_ROLE_MODEL_REPORT_DIR="+filepath.Join(dir, "reports"),
				"MYSQL_HOST=unused", "MYSQL_USERNAME=unused", "MYSQL_PASSWORD=unused", "MYSQL_DBNAME=unused", "QS_MYSQL_DBNAME=")
			output, err := cmd.CombinedOutput()
			if (err == nil) != (state == "applied_unchanged") {
				t.Fatalf("state=%s err=%v output=%s", state, err, output)
			}
			data, err := os.ReadFile(calls)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "apply") || strings.Contains(string(data), "preflight") {
				t.Fatalf("repeated migration: %s", data)
			}
			if state != "applied_unchanged" && string(data) != "status\n" {
				t.Fatalf("mutated drifted facts: %s", data)
			}
		})
	}
}
