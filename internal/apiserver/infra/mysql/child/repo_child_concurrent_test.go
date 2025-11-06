package child

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/child"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	m "github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 并发创建相同的儿童档案（相同身份证号），期望只有 1 条记录被写入，
// 其余并发请求因唯一约束被 translator 映射为业务错误 code.ErrIdentityChildExists。
func TestChildRepository_Create_ConcurrentDuplicateDetection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&ChildPO{}))

	repo := NewRepository(db)
	ctx := context.Background()

	const concurrency = 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errs := make(chan error, concurrency)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	idNumber := "ID202511060001"
	for i := 0; i < concurrency; i++ {
		delay := rng.Intn(8)
		go func(d int) {
			defer wg.Done()
			// tiny jitter to reduce lock storms on SQLite
			time.Sleep(time.Millisecond * time.Duration(d))
			c, err := domain.NewChild("Alice", domain.WithIDCard(m.NewIDCard("Alice", idNumber)))
			if err != nil {
				errs <- err
				return
			}
			if err := repo.Create(ctx, c); err != nil {
				errs <- err
				return
			}
			errs <- nil
		}(delay)
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
			if perrors.IsCode(ue, code.ErrIdentityChildExists) {
				mappedCount++
				break
			}
			ue = errors.Unwrap(ue)
		}
	}

	require.Equal(t, 1, success, "only one create should succeed")
	require.GreaterOrEqual(t, mappedCount, 1, "at least one error should be mapped to ErrIdentityChildExists")

	var cnt int64
	require.NoError(t, db.Model(&ChildPO{}).
		Where("id_card = ?", idNumber).
		Count(&cnt).Error)
	require.Equal(t, int64(1), cnt)
}
