package scopemigrate

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	"time"
)

type receiptResult struct {
	MigrationID, Fingerprint, AfterHash, Status string
	CreatedAt, UpdatedAt                        time.Time
	ActorID, RollbackActorID, RollbackHash      string
}

func receiptResultCodec(result **Receipt) *operation.ResultCodec {
	return operation.ProjectResult("iam.maintenance.scopemigrate.receipt/v1", func() (receiptResult, error) {
		v := *result
		if v == nil {
			return receiptResult{}, operation.ErrInvalid
		}
		return receiptResult{MigrationID: v.MigrationID, Fingerprint: v.Fingerprint, AfterHash: v.AfterHash, Status: v.Status, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, ActorID: v.ActorID, RollbackActorID: v.RollbackActorID, RollbackHash: v.RollbackHash}, nil
	}, func(saved receiptResult) error {
		v := Receipt{MigrationID: saved.MigrationID, Fingerprint: saved.Fingerprint, AfterHash: saved.AfterHash, Status: saved.Status, CreatedAt: saved.CreatedAt, UpdatedAt: saved.UpdatedAt, ActorID: saved.ActorID, RollbackActorID: saved.RollbackActorID, RollbackHash: saved.RollbackHash}
		*result = &v
		return nil
	})
}
