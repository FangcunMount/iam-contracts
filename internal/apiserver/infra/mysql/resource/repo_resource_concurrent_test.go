package resource

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 并发创建相同的 resource（相同 key），期望只有 1 条记录被写入，
// 其余并发请求因唯一约束被 translator 映射为业务错误 code.ErrResourceAlreadyExists。
func TestResourceRepository_Create_ConcurrentDuplicateDetection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&ResourcePO{}))

	repo := NewResourceRepository(db)
	ctx := context.Background()

	const concurrency = 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errs := make(chan error, concurrency)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			// add tiny random delay to reduce SQLITE table-lock contention
			time.Sleep(time.Millisecond * time.Duration(rng.Intn(8)))
			r := domain.NewResource("app:domain:type:dup", []string{"read", "write"})
			if err := repo.Create(ctx, &r); err != nil {
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
		if e != nil {
			t.Logf("err: %T: %v", e, e)
		}
		if e == nil {
			success++
			continue
		}

		var ue error = e
		for ue != nil {
			if perrors.IsCode(ue, code.ErrResourceAlreadyExists) {
				mappedCount++
				break
			}
			ue = errors.Unwrap(ue)
		}
	}

	require.Equal(t, 1, success, "only one create should succeed")
	require.GreaterOrEqual(t, mappedCount, 1, "at least one error should be mapped to ErrResourceAlreadyExists")

	var cnt int64
	require.NoError(t, db.Model(&ResourcePO{}).
		Where("`key` = ?", "app:domain:type:dup").
		Count(&cnt).Error)
	require.Equal(t, int64(1), cnt)
}
