// Package testutil contains explicitly labeled planning and native-root fixtures.
// Probe is source-only; integration-tagged NativeWriter uses original MySQL
// transactions with stored half receipts and never claims actual Broker proof.
package testutil

import (
	"context"
	"database/sql"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	database "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"gorm.io/gorm"
)

type Probe struct {
	event.Stager
	Requests      []policywrite.Request
	Writes        int
	BeforePreview func()
}

func (p *Probe) WithinPolicyWrite(ctx context.Context, db *gorm.DB, r policywrite.Request, fn func(context.Context) error) error {
	p.Requests = append(p.Requests, r)
	if p.BeforePreview != nil {
		p.BeforePreview()
	}
	var changed bool
	err := database.NewUnitOfWork(db).WithinTransaction(ctx, func(readCtx context.Context) error {
		tx, err := database.RequireTx(readCtx)
		if err != nil {
			return err
		}
		changed, err = r.Preview(readCtx, tx)
		return err
	}, database.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil || !changed {
		return err
	}
	p.Writes++
	return database.NewUnitOfWork(db).WithinTransaction(ctx, fn)
}
