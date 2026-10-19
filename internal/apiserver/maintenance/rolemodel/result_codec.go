package rolemodel

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	"time"
)

type receiptResult struct {
	MigrationID, Fingerprint, AfterHash, Status string
	CreatedAt, UpdatedAt                        time.Time
}

func receiptResultCodec(result *Receipt) *operation.ResultCodec {
	return operation.ProjectResult("iam.maintenance.rolemodel.receipt/v1", func() (receiptResult, error) {
		v := result
		if v == nil {
			return receiptResult{}, operation.ErrInvalid
		}
		return receiptResult{MigrationID: v.MigrationID, Fingerprint: v.Fingerprint, AfterHash: v.AfterHash, Status: v.Status, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}, nil
	}, func(saved receiptResult) error {
		v := Receipt{MigrationID: saved.MigrationID, Fingerprint: saved.Fingerprint, AfterHash: saved.AfterHash, Status: saved.Status, CreatedAt: saved.CreatedAt, UpdatedAt: saved.UpdatedAt}
		*result = v
		return nil
	})
}
