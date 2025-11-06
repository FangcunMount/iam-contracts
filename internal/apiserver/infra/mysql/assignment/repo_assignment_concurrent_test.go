package assignment

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 并发创建相同的 assignment（相同 subject_type+subject_id+role_id+tenant_id），
// 在测试环境为表添加唯一索引以触发重复错误，期望只有 1 条记录写入，
// 其余被翻译为 code.ErrAssignmentAlreadyExists。
func TestAssignmentRepository_Create_ConcurrentDuplicateDetection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&AssignmentPO{}))

	// 为测试环境显式创建唯一索引，避免在 PO tag 中改动生产 schema
	// 复合唯一键: subject_type, subject_id, role_id, tenant_id
	_ = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_assignment ON iam_authz_assignments(subject_type, subject_id, role_id, tenant_id)")

	repo := NewAssignmentRepository(db)
	ctx := context.Background()

	const concurrency = 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errs := make(chan error, concurrency)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond * time.Duration(rng.Intn(8)))
			a := domain.NewAssignment(domain.SubjectTypeUser, "user-123", 42, "tenant-1")
			if err := repo.Create(ctx, &a); err != nil {
				errs <- err
				return
			}
			errs <- nil
		}()
	}

	wg.Wait()
	close(errs)

	var success int
	var mappedCount int
	for e := range errs {
		if e == nil {
			success++
			continue
		}

		var ue error = e
		for ue != nil {
			if perrors.IsCode(ue, code.ErrAssignmentAlreadyExists) {
				mappedCount++
				break
			}
			ue = errors.Unwrap(ue)
		}
	}

	require.Equal(t, 1, success, "only one create should succeed")
	require.GreaterOrEqual(t, mappedCount, 1, "at least one error should be mapped to ErrAssignmentAlreadyExists")

	var cnt int64
	require.NoError(t, db.Model(&AssignmentPO{}).
		Where("subject_type = ? AND subject_id = ? AND role_id = ? AND tenant_id = ?", "user", "user-123", 42, "tenant-1").
		Count(&cnt).Error)
	require.Equal(t, int64(1), cnt)
}
