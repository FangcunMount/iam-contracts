package policy

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	dbmysql "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
)

var _ domain.PlannedRepository = (*PolicyVersionRepository)(nil)

// IncrementExpected requires the original live SQL root. The host must already
// hold its SDK execution guard, and must roll the whole root back on any error.
// It does not read a new target, open another transaction or initialize a row.
func (r *PolicyVersionRepository) IncrementExpected(ctx context.Context, expected int64, by, reason string, at time.Time) (*domain.PolicyVersion, error) {
	if r == nil || r.db == nil || r.db.Error != nil || r.db.Statement == nil || r.db.Dialector == nil || r.db.Dialector.Name() != "mysql" {
		return nil, dbmysql.ErrActiveTransactionRequired
	}
	pool := r.db.Statement.ConnPool
	if prepared, ok := pool.(*gorm.PreparedStmtTX); ok {
		pool = prepared.Tx
	}
	if root, ok := pool.(*sql.Tx); !ok || root == nil {
		return nil, dbmysql.ErrActiveTransactionRequired
	}
	if expected < 1 || expected == math.MaxInt64 || at.IsZero() || strings.TrimSpace(by) == "" || !utf8.ValidString(by) || !utf8.ValidString(reason) || utf8.RuneCountInString(by) > 64 || utf8.RuneCountInString(reason) > 512 {
		return nil, domain.ErrInvalidVersionPlan
	}
	// UpdateColumns avoids allocating another clock value in ORM audit hooks.
	result := r.db.WithContext(ctx).Model(&PolicyVersionPO{}).Where("id = ? AND policy_version = ? AND deleted_at IS NULL", 1, expected).UpdateColumns(map[string]any{"policy_version": expected + 1, "changed_by": by, "reason": reason, "updated_at": at, "updated_by": dbmysql.UserIDOrZero(ctx).Uint64()})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, domain.ErrStaleVersion
	}
	value := domain.NewPolicyVersion(expected+1, domain.WithID(domain.PolicyVersionID(meta.FromUint64(1))), domain.WithChangedBy(by), domain.WithReason(reason))
	return &value, nil
}
