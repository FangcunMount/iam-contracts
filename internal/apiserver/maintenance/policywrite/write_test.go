package policywrite_test

import (
	"context"
	"errors"
	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	"github.com/FangcunMount/iam/v5/internal/apiserver/testhelpers"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type refusingWriter struct{ err error }

func (w refusingWriter) Stage(context.Context, ...event.DomainEvent) error {
	return errors.New("unexpected stage")
}
func (w refusingWriter) WithinPolicyWrite(context.Context, *gorm.DB, policywrite.Request, func(context.Context) error) error {
	return w.err
}
func TestConfiguredPolicyWriterNeverFallsBack(t *testing.T) {
	db := testhelpers.SetupTempSQLiteDB(t)
	sentinel := errors.New("admission closed")
	req := policywrite.Request{Operation: "maintenance.test", Preview: func(context.Context, *gorm.DB) (bool, error) { t.Fatal("unexpected preview"); return false, nil }}
	require.ErrorIs(t, policywrite.Within(context.Background(), db, refusingWriter{sentinel}, req, func(context.Context) error { t.Fatal("unexpected business transaction"); return nil }), sentinel)
	_, err := req.AuthorizationRequest().Preview(context.Background(), app.TxRepositories{})
	require.Error(t, err)
}
