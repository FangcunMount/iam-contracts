//go:build integration

package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	roleapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	authzcontainer "github.com/FangcunMount/iam/v5/internal/apiserver/container/authz"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	journal "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/authzoperation"
	policyrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	host "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/uow/authz"
	authzhttp "github.com/FangcunMount/iam/v5/internal/apiserver/transport/rest/authz"
	"github.com/FangcunMount/iam/v5/internal/apiserver/transport/rest/authz/handler"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/iam/v5/internal/pkg/requestctx"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type nativeOperationChecker func(context.Context, authorization.Request) (authorization.Decision, error)

func (f nativeOperationChecker) Check(ctx context.Context, r authorization.Request) (authorization.Decision, error) {
	return f(ctx, r)
}

// Actual REST/app/host-journal/SDK-store integration. Authentication is an
// explicit verified-user fixture, Broker is the named stored-half fixture; this
// test is neither a system-account login nor a real Broker/network proof.
func testM3OperationHTTPNative(t *testing.T, db *gorm.DB, binding transaction.Binding) {
	local, err := host.NewTransactionUnitOfWork(db, nativeSubjects{}, binding)
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	records, err := journal.New(pool, binding)
	require.NoError(t, err)
	sender := &nativeStoredHalfSender{store: local.Store()}
	topic := "iam.authz.version.v2"
	physical, err := transport.PhysicalTopicV2(topic, transport.TransactionMessageType, binding.Owner, "m3-1")
	require.NoError(t, err)
	writer, err := host.NewPlannedUnitOfWork(local, sender, host.PlanConfig{LogicalTopic: topic, PhysicalTopic: physical, Sends: 8, ProofBudget: 200 * time.Millisecond, Admission: func() bool { return true }, Journal: records, OperationLease: time.Minute})
	require.NoError(t, err)
	t.Cleanup(func() { writer.StopAdmission(); require.NoError(t, writer.Wait(context.Background())) })
	allowed, stale := true, false
	guard := management.NewGuard(nativeOperationChecker(func(ctx context.Context, _ authorization.Request) (authorization.Decision, error) {
		version, err := policyrepo.NewPolicyVersionRepository(db).GetCurrent(ctx)
		if err != nil {
			return authorization.Decision{}, err
		}
		cached := version.Version
		if stale {
			cached--
		}
		return authorization.Decision{Allowed: allowed, PolicyVersion: cached}, nil
	}))
	query, err := writer.NewOperationQuery(authzcontainer.NewOperationAccessPolicy(writer, guard))
	require.NoError(t, err)
	service := roleapp.NewRoleCatalog(writer, nil, guard)
	engine := gin.New()
	actor := meta.ID(7)
	authenticated := true
	authzhttp.Register(engine, authzhttp.Dependencies{RoleHandler: handler.NewRoleHandler(service, nil), OperationHandler: handler.NewOperationHandler(query), AuthMiddleware: func(c *gin.Context) {
		if !authenticated {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		user, err := subject.NewUserRef(actor)
		require.NoError(t, err)
		requestctx.SetUserID(c, actor)
		c.Request = c.Request.WithContext(management.WithAuthenticatedUser(c.Request.Context(), user))
		c.Next()
	}, Permission: func(resource, action string) gin.HandlerFunc {
		return func(c *gin.Context) {
			if err := guard.RequireOperation(c.Request.Context(), resource, action); err != nil {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
		}
	}})
	var active *httptest.ResponseRecorder
	request := func(method, path, key, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		active = httptest.NewRecorder()
		engine.ServeHTTP(active, r)
		return active
	}
	body := `{"name":"m3:http-receipt","display_name":"First result","description":"public"}`
	first := request("POST", "/api/v4/authz/roles", "http-original-key", body)
	require.Equal(t, 200, first.Code, first.Body.String())
	id := first.Header().Get("Operation-ID")
	require.True(t, operation.ValidID(id))
	location := first.Header().Get("Operation-Location")
	require.Equal(t, "/api/v4/authz/operations/"+id, location)
	calls := sender.calls
	again := request("POST", "/api/v4/authz/roles", "http-original-key", body)
	require.Equal(t, 200, again.Code, again.Body.String())
	require.JSONEq(t, first.Body.String(), again.Body.String())
	require.Equal(t, id, again.Header().Get("Operation-ID"))
	require.Equal(t, calls, sender.calls)
	conflict := request("POST", "/api/v4/authz/roles", "http-original-key", strings.Replace(body, "First result", "Other input", 1))
	require.Equal(t, 409, conflict.Code)
	require.Equal(t, calls, sender.calls)
	status := request("GET", location, "", "")
	require.Equal(t, 200, status.Code, status.Body.String())
	var response struct {
		Data operation.QueryView `json:"data"`
	}
	require.NoError(t, json.Unmarshal(status.Body.Bytes(), &response))
	require.Equal(t, operation.Committed, response.Data.State)
	require.Equal(t, transaction.LocalCommitted, response.Data.Phases.Local)
	require.Equal(t, transaction.FinalUnknown, response.Data.Phases.Final, "stored half fixture never confirms Broker final RPC")
	for _, private := range []string{"http-original-key", "sdk_plan", "execution_token", "BrokerTransactionID", "candidate"} {
		require.NotContains(t, status.Body.String(), private)
	}
	actor = 8
	require.Equal(t, 404, request("GET", location, "", "").Code)
	actor = 7
	allowed = false
	require.Equal(t, 403, request("GET", location, "", "").Code)
	allowed = true
	stale = true
	require.Equal(t, 403, request("GET", location, "", "").Code, "stale ALLOW reused")
	stale = false
	authenticated = false
	require.Equal(t, 401, request("GET", location, "", "").Code)
	authenticated = true

	// A competitor changes the policy version after read-only no-op planning but
	// before its receipt root. The receipt must abort without publishing/mutating.
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &created))
	require.NotEmpty(t, created.Data.ID)
	require.NoError(t, db.Exec("CREATE TRIGGER iam_m3_noop_drift AFTER INSERT ON iam_authz_operations FOR EACH ROW UPDATE authz_policy_versions SET policy_version=policy_version+1 WHERE id=1").Error)
	dropTrigger := func() { require.NoError(t, db.Exec("DROP TRIGGER IF EXISTS iam_m3_noop_drift").Error) }
	t.Cleanup(dropTrigger)
	noopCalls := sender.calls
	drifted := request("PUT", "/api/v4/authz/roles/"+created.Data.ID, "http-noop-drift", `{"display_name":"First result","description":"public"}`)
	dropTrigger()
	require.Equal(t, 409, drifted.Code, drifted.Body.String())
	require.Equal(t, noopCalls, sender.calls, "no-op conflict prepared a half message")
	noOpRecord, err := records.Read(t.Context(), drifted.Header().Get("Operation-ID"))
	require.NoError(t, err)
	require.False(t, noOpRecord.Candidate.Changed())
	require.Equal(t, journal.Aborted, noOpRecord.State)
	require.Empty(t, noOpRecord.Result)

	// Hold the exact business receipt row so authoritative abort cannot complete.
	// A local error/cancellation alone cannot become a rollback proof.
	var held *sql.Tx
	t.Cleanup(func() {
		if held != nil {
			_ = held.Rollback()
		}
	})
	sender.beforeLocal = func() error {
		var err error
		held, err = pool.BeginTx(t.Context(), nil)
		if err != nil {
			return err
		}
		var state string
		err = held.QueryRowContext(t.Context(), "SELECT state FROM iam_authz_operations WHERE operation_id=? FOR UPDATE", active.Header().Get("Operation-ID")).Scan(&state)
		if err != nil {
			return err
		}
		return errors.New("explicit stored-half fixture failure before local execution")
	}
	unknown := request("POST", "/api/v4/authz/roles", "http-unknown-key", strings.Replace(body, "http-receipt", "http-unknown", 1))
	require.Equal(t, 503, unknown.Code, unknown.Body.String())
	require.Contains(t, unknown.Body.String(), "operation_outcome_unknown")
	unknownID := unknown.Header().Get("Operation-ID")
	require.True(t, operation.ValidID(unknownID))
	require.NotContains(t, unknown.Body.String(), "fixture failure")
	require.NotNil(t, held)
	require.NoError(t, held.Rollback())
	sender.beforeLocal = nil
	prepared, err := records.Read(t.Context(), unknownID)
	require.NoError(t, err)
	require.Contains(t, []journal.State{journal.Prepared, journal.Aborted}, prepared.State)
	unknownCalls := sender.calls
	duplicate := request("POST", "/api/v4/authz/roles", "http-unknown-key", strings.Replace(body, "http-receipt", "http-unknown", 1))
	require.Contains(t, []int{503, 409}, duplicate.Code)
	if duplicate.Code == 409 {
		require.Contains(t, duplicate.Body.String(), "operation_aborted")
	} else {
		require.Contains(t, duplicate.Body.String(), "operation_outcome_unknown")
	}
	require.Equal(t, unknownCalls, sender.calls, "unknown operation automatically reexecuted")
	var count int64
	require.NoError(t, db.Table("authz_roles").Where("name=?", "m3:http-unknown").Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Exec("UPDATE iam_authz_operations SET lease_until=TIMESTAMPADD(SECOND,-1,UTC_TIMESTAMP(6)) WHERE operation_id=? AND state='prepared'", unknownID).Error)
	_, err = records.AbortExpired(t.Context(), 100)
	require.NoError(t, err)
	aborted, err := records.Read(t.Context(), unknownID)
	require.NoError(t, err)
	require.Equal(t, journal.Aborted, aborted.State)
}
