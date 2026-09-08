package policy

import (
	"context"
	"math"
	"sync"
	"testing"

	testutil "github.com/FangcunMount/iam/v5/internal/apiserver/application/identity/testutil"
	testhelpers "github.com/FangcunMount/iam/v5/internal/apiserver/testhelpers"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPolicyVersionRepositoryConcurrentInitializationCreatesOneRow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	require.NoError(t, db.AutoMigrate(&PolicyVersionPO{}))
	repo := NewPolicyVersionRepository(db)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- testhelpers.RetryOnDBLocked(func() error { _, err := repo.GetOrCreate(context.Background()); return err })
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var count int64
	require.NoError(t, db.Model(&PolicyVersionPO{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestPolicyVersionIncrementIsAtomicWithCallerTransaction(t *testing.T) {
	db := testutil.SetupTestDB(t)
	require.NoError(t, db.AutoMigrate(&PolicyVersionPO{}))
	ctx := context.Background()
	repo := NewPolicyVersionRepository(db)
	_, err := repo.GetOrCreate(ctx)
	require.NoError(t, err)
	for i := int64(2); i <= 9; i++ {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			v, err := NewPolicyVersionRepository(tx).Increment(ctx, "operator", "test")
			if err == nil {
				require.Equal(t, i, v.Version)
			}
			return err
		}))
	}
	sentinel := gorm.ErrInvalidTransaction
	require.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := NewPolicyVersionRepository(tx).Increment(ctx, "operator", "rollback")
		require.NoError(t, err)
		return sentinel
	}), sentinel)
	current, err := repo.GetCurrent(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(9), current.Version)
	require.NoError(t, db.Model(&PolicyVersionPO{}).Where("id = ?", 1).Update("policy_version", math.MaxInt64).Error)
	_, err = repo.Increment(ctx, "operator", "overflow")
	require.Error(t, err)
	current, err = repo.GetCurrent(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(math.MaxInt64), current.Version)
}
