//go:build integration

package migration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	assignmentapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	grantapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/permissiongrant"
	resourceapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/resource"
	roleapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	hostauthz "github.com/FangcunMount/iam/v5/internal/apiserver/container/authz"
	"github.com/FangcunMount/iam/v5/internal/apiserver/container/platform"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/subjectresolver"
	grantrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	rolerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/FangcunMount/reliable-messaging/transport/rocketmq"
	driver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fanoutReadOnlyStager struct{}

func (fanoutReadOnlyStager) Stage(context.Context, ...event.DomainEvent) error {
	return errors.New("consumer-only fixture cannot write")
}

type fanoutUserResolver struct{}

func (fanoutUserResolver) ResolveUser(_ context.Context, id meta.ID) error {
	if id != meta.ID(7) {
		return errors.New("fixture identity outside declared subject")
	}
	return nil
}

// Optional in the original-root profile, required by the explicit fanout
// profile. The complete IAM modules and QS test binary use independent groups.
func proveM3BusinessFanout(t *testing.T, pool *sql.DB, db *gorm.DB, owner *platform.TransactionEventing) {
	t.Helper()
	ctx := management.WithAuthenticatedService(t.Context(), "admin")
	resolver := assignment.NewSubjectResolverRegistry(subjectresolver.NewUserSubjectResolver(fanoutUserResolver{}))
	writer := owner.Writer.ApplicationUnitOfWork(resolver)
	roles := roleapp.NewRoleCatalog(writer, nil, management.NewGuard(nil))
	cmd, err := roleapp.NewCreateRoleCommand("m3:fanout_reader", "Fanout reader", "")
	require.NoError(t, err)
	cmd.ChangedBy = "fanout-fixture"
	role, err := roles.CreateRole(ctx, cmd)
	require.NoError(t, err)
	resources := resourceapp.NewResourceCatalog(writer, nil, nativeCatalogWriter{})
	actor, err := subject.NewUserRef(meta.ID(7))
	require.NoError(t, err)
	resourceCmd, err := resourceapp.NewCreateResourceCommand("m3:test:collection:fanout", "Fanout resource", "", "", "", []string{"read"}, "")
	require.NoError(t, err)
	resourceCmd.Actor = actor
	resourceCmd.ChangedBy = "fanout-fixture"
	resource, err := resources.CreateResource(ctx, resourceCmd)
	require.NoError(t, err)
	grants := grantapp.NewService(writer, grantrepo.NewRepository(db), nil, management.NewGuard(nil))
	grant, err := grants.Create(ctx, grantapp.CreateCommand{RoleID: role.ID, ResourceID: resource.ID, Action: "read", GrantedBy: "fanout-fixture"})
	require.NoError(t, err)
	roleRepository := rolerepo.NewRoleRepository(db)
	assignments := assignmentapp.NewCommandService(assignment.NewValidatorWithSubjectResolver(roleRepository, resolver), roleRepository, writer, nil, management.NewGuard(nil))
	_, err = assignments.Grant(ctx, assignmentapp.GrantCommand{SubjectType: assignment.SubjectTypeUser, SubjectID: meta.ID(7), RoleID: role.ID, GrantedBy: "fanout-fixture"})
	require.NoError(t, err)
	request, err := authorization.NewRequest(actor, resource.KeyString(), "read")
	require.NoError(t, err)
	modules := make([]*hostauthz.AuthzModule, 2)
	var unexpected atomic.Bool
	for i, key := range []string{"IAM_M3_FANOUT_A_CONFIG", "IAM_M3_FANOUT_B_CONFIG"} {
		modules[i] = hostauthz.NewAuthzModule()
		require.NoError(t, modules[i].InitializeWithDeps(hostauthz.AuthzModuleDeps{DB: db, EventStager: fanoutReadOnlyStager{}, UserResolver: fanoutUserResolver{}, AssignmentConstraintsFile: "../../../configs/grpc_assignment_constraints.yaml"}))
		opts := readM3BrokerOptions(t, key)
		config, err := opts.RocketMQ([]string{"iam.authz.version.v2"}, 2, 5*time.Second)
		require.NoError(t, err)
		sub, err := rocketmq.NewSubscriber(rocketmq.SubscriberConfig{Config: config, MaxAttempts: 3, InvisibleDuration: 15 * time.Second, HandlerTimeout: 5 * time.Second, RetryDelay: time.Second})
		require.NoError(t, err)
		channel := []string{"iam-m3-native-fanout-a", "iam-m3-native-fanout-b"}[i]
		syncer := modules[i].RocketMQPolicySyncSubscriber(sub, func(context.Context, transport.Failure) error {
			unexpected.Store(true)
			return errors.New("unexpected fanout terminal delivery")
		}, channel)
		require.NotNil(t, syncer)
		t.Cleanup(func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			require.NoError(t, syncer.StopWithContext(closeCtx))
		})
		require.NoError(t, syncer.Start(t.Context()))
		decision, err := modules[i].ApplicationCapabilities().AuthorizationDecisions.Check(t.Context(), request)
		require.NoError(t, err)
		require.True(t, decision.Allowed, "fixture must demonstrate a permission before revocation")
	}

	// The QS child owns only its read pool and actual QS SDK consumer. Its
	// primary-reader test port is deliberately distinct from production gRPC.
	directory := t.TempDir()
	require.NoError(t, os.Chmod(directory, 0700))
	cfg, err := driver.ParseDSN(os.Getenv("IAM_M3_TEST_MYSQL_DSN"))
	require.NoError(t, err)
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT DATABASE()").Scan(&cfg.DBName))
	ready, target, receipt := filepath.Join(directory, "ready.json"), filepath.Join(directory, "target.json"), filepath.Join(directory, "receipt.json")
	input, _ := json.Marshal(map[string]string{"dsn": cfg.FormatDSN(), "config": os.Getenv("IAM_M3_FANOUT_QS_CONFIG"), "ready": ready, "target": target, "receipt": receipt})
	inputPath := filepath.Join(directory, "input.json")
	require.NoError(t, os.WriteFile(inputPath, input, 0600))
	binary := os.Getenv("IAM_M3_FANOUT_QS_BINARY")
	require.True(t, filepath.IsAbs(binary), "fixed QS child binary required")
	child := exec.Command(binary, "-test.run=^TestM3RocketMQIAMFanoutChild$", "-test.count=1", "-test.timeout=90s", "-test.v")
	child.Env = append(os.Environ(), "QS_M3_FANOUT_INPUT="+inputPath)
	output, err := os.OpenFile(filepath.Join(directory, "child-private.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	require.NoError(t, err)
	child.Stdout = output
	child.Stderr = output
	require.NoError(t, child.Start())
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	joined := false
	t.Cleanup(func() {
		if !joined {
			_ = child.Process.Kill()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Error("QS child process not drained")
			}
		}
		if output.Close() != nil {
			t.Error("private child log not closed")
		}
		if t.Failed() {
			raw, readErr := os.ReadFile(filepath.Join(directory, "child-private.log"))
			if readErr == nil {
				// Stack/assertion locations only, never copy runtime errors or values.
				locations := regexp.MustCompile(`(?m)^\s*(?:[^\s\n]*/)?([A-Za-z0-9_.-]+\.go:[0-9]+)(?: \+0x[0-9a-f]+|:).*$`).FindAllSubmatch(raw, 64)
				for _, location := range locations {
					t.Logf("M3_QS_FAILURE_SITE %s", location[1])
				}
				if bytes.Contains(raw, []byte("WARNING: DATA RACE")) {
					t.Log("M3_QS_FAILURE_RACE")
				}
			}
		}
	})
	waitCtx, cancel := context.WithTimeout(t.Context(), 75*time.Second)
	defer cancel()
	waitFile := func(path string) {
		t.Helper()
		for {
			if _, err := os.Stat(path); err == nil {
				return
			} else if !os.IsNotExist(err) {
				t.Fatal("child proof read failed")
			}
			select {
			case err := <-done:
				joined = true
				t.Fatalf("QS child exited before proof: %v", err)
			case <-waitCtx.Done():
				t.Fatal("child proof deadline")
			case <-time.After(20 * time.Millisecond):
			}
		}
	}
	waitFile(ready)
	require.NoError(t, grants.Revoke(ctx, grantapp.RevokeCommand{GrantID: grant.ID, RevokedBy: "fanout-fixture"}))
	var version int64
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT policy_version FROM authz_policy_versions WHERE id=1").Scan(&version))
	targetBody, _ := json.Marshal(version)
	require.NoError(t, os.WriteFile(target, targetBody, 0600))
	require.Eventually(t, func() bool {
		for _, module := range modules {
			health := module.ApplicationCapabilities().RuntimeHealth.RuntimeHealthDetails()
			eventVersion, _ := health["last_event_version"].(int64)
			loadedVersion, _ := health["loaded_policy_version"].(int64)
			if eventVersion < version || loadedVersion < version {
				return false
			}
			decision, err := module.ApplicationCapabilities().AuthorizationDecisions.Check(t.Context(), request)
			if err != nil || decision.Allowed || int64(decision.PolicyVersion) < version {
				return false
			}
		}
		return true
	}, 45*time.Second, 25*time.Millisecond, "both independent IAM subscriptions must observe the committed revocation")
	waitFile(receipt)
	var qs struct {
		Notification bool  `json:"notification_received"`
		Evicted      bool  `json:"both_subjects_evicted"`
		OldRejected  bool  `json:"old_snapshot_rejected"`
		Primary      bool  `json:"primary_proof_verified"`
		Drained      bool  `json:"subscriber_drained"`
		Version      int64 `json:"target_version"`
	}
	raw, err := os.ReadFile(receipt)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &qs))
	require.True(t, qs.Notification && qs.Evicted && qs.OldRejected && qs.Primary && qs.Drained && qs.Version == version)
	select {
	case err := <-done:
		joined = true
		require.NoError(t, err)
	case <-waitCtx.Done():
		t.Fatal("QS child did not exit")
	}
	require.False(t, unexpected.Load())
	t.Log(`M3_FANOUT_RECEIPT {"iam_instances":2,"iam_revocation_observed":true,"qs_notification_received":true,"qs_both_subjects_evicted":true,"qs_primary_proof_verified":true,"qs_subscriber_drained":true,"failed":0,"skipped":0}`)
}
