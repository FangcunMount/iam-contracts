package guardianship

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	childdomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/child"
	userdomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

type stubGuardianshipRepo struct {
	childrenResults map[uint64][]*Guardianship
	findErr         error
	findCalls       int
}

func (s *stubGuardianshipRepo) Create(context.Context, *Guardianship) error { return nil }
func (s *stubGuardianshipRepo) FindByID(context.Context, idutil.ID) (*Guardianship, error) {
	return nil, nil
}
func (s *stubGuardianshipRepo) FindByChildID(ctx context.Context, id meta.ID) ([]*Guardianship, error) {
	s.findCalls++
	if s.findErr != nil {
		return nil, s.findErr
	}
	if s.childrenResults == nil {
		return nil, nil
	}
	return s.childrenResults[id.Uint64()], nil
}
func (s *stubGuardianshipRepo) FindByUserID(context.Context, meta.ID) ([]*Guardianship, error) {
	return nil, nil
}
func (s *stubGuardianshipRepo) FindByUserIDAndChildID(context.Context, meta.ID, meta.ID) (*Guardianship, error) {
	return nil, nil
}
func (s *stubGuardianshipRepo) IsGuardian(context.Context, meta.ID, meta.ID) (bool, error) {
	return false, nil
}
func (s *stubGuardianshipRepo) Update(context.Context, *Guardianship) error { return nil }

type stubChildDomainRepo struct {
	child *childdomain.Child
	err   error
}

func (s *stubChildDomainRepo) Create(context.Context, *childdomain.Child) error { return nil }
func (s *stubChildDomainRepo) FindByID(context.Context, meta.ID) (*childdomain.Child, error) {
	return s.child, s.err
}
func (s *stubChildDomainRepo) FindByName(context.Context, string) (*childdomain.Child, error) {
	return nil, nil
}
func (s *stubChildDomainRepo) FindByIDCard(context.Context, meta.IDCard) (*childdomain.Child, error) {
	return nil, nil
}
func (s *stubChildDomainRepo) FindListByName(context.Context, string) ([]*childdomain.Child, error) {
	return nil, nil
}
func (s *stubChildDomainRepo) FindListByNameAndBirthday(context.Context, string, meta.Birthday) ([]*childdomain.Child, error) {
	return nil, nil
}
func (s *stubChildDomainRepo) FindSimilar(context.Context, string, meta.Gender, meta.Birthday) ([]*childdomain.Child, error) {
	return nil, nil
}
func (s *stubChildDomainRepo) Update(context.Context, *childdomain.Child) error { return nil }

type stubUserDomainRepo struct {
	user *userdomain.User
	err  error
}

func (s *stubUserDomainRepo) Create(context.Context, *userdomain.User) error { return nil }
func (s *stubUserDomainRepo) FindByID(context.Context, meta.ID) (*userdomain.User, error) {
	return s.user, s.err
}
func (s *stubUserDomainRepo) FindByPhone(context.Context, meta.Phone) (*userdomain.User, error) {
	return nil, nil
}
func (s *stubUserDomainRepo) Update(context.Context, *userdomain.User) error { return nil }

func TestGuardianshipManager_AddGuardianSuccess(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.FromUint64(1)}}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.FromUint64(2)}}
	guardRepo := &stubGuardianshipRepo{childrenResults: make(map[uint64][]*Guardianship)}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1), RelParent)

	require.NoError(t, err)
	require.NotNil(t, guardian)
	assert.Equal(t, meta.FromUint64(2), guardian.User)
	assert.Equal(t, meta.FromUint64(1), guardian.Child)
	assert.Equal(t, RelParent, guardian.Rel)
	assert.False(t, guardian.EstablishedAt.IsZero())
}

func TestGuardianshipManager_AddGuardian_Duplicate(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.FromUint64(1)}}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.FromUint64(2)}}
	existing := &Guardianship{User: meta.FromUint64(2), Child: meta.FromUint64(1)}
	guardRepo := &stubGuardianshipRepo{
		childrenResults: map[uint64][]*Guardianship{
			1: {existing},
		},
	}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "guardian already exists")
}

func TestGuardianshipManager_AddGuardian_ChildNotFound(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: nil}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.FromUint64(2)}}
	guardRepo := &stubGuardianshipRepo{}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "child not found")
}

func TestGuardianshipManager_AddGuardian_UserRepoError(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.FromUint64(1)}}
	userRepo := &stubUserDomainRepo{err: errors.New("db error")}
	guardRepo := &stubGuardianshipRepo{}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "find user failed")
}

func TestGuardianshipManager_AddGuardian_FindByChildError(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.FromUint64(1)}}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.FromUint64(2)}}
	guardRepo := &stubGuardianshipRepo{findErr: errors.New("db error")}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "find guardians failed")
}

func TestGuardianshipManager_RemoveGuardianSuccess(t *testing.T) {
	target := &Guardianship{User: meta.FromUint64(2), Child: meta.FromUint64(1)}
	guardRepo := &stubGuardianshipRepo{
		childrenResults: map[uint64][]*Guardianship{
			1: {target},
		},
	}
	manager := NewManagerService(guardRepo, &stubChildDomainRepo{}, &stubUserDomainRepo{})

	removed, err := manager.RemoveGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1))

	require.NoError(t, err)
	assert.NotNil(t, removed)
	assert.NotNil(t, removed.RevokedAt)
	assert.True(t, removed.RevokedAt.After(time.Time{}))
}

func TestGuardianshipManager_RemoveGuardian_NotFound(t *testing.T) {
	guardRepo := &stubGuardianshipRepo{
		childrenResults: map[uint64][]*Guardianship{
			1: {},
		},
	}
	manager := NewManagerService(guardRepo, &stubChildDomainRepo{}, &stubUserDomainRepo{})

	removed, err := manager.RemoveGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1))

	require.Error(t, err)
	assert.Nil(t, removed)
	assert.Contains(t, fmt.Sprintf("%-v", err), "active guardian not found")
}

func TestGuardianshipManager_AddGuardian_ChildRepoError(t *testing.T) {
	childRepo := &stubChildDomainRepo{err: errors.New("db error")}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.FromUint64(2)}}
	guardRepo := &stubGuardianshipRepo{}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "find child failed")
}

func TestGuardianshipManager_AddGuardian_UserNotFound(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.FromUint64(1)}}
	userRepo := &stubUserDomainRepo{user: nil}
	guardRepo := &stubGuardianshipRepo{}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "user not found")
}

func TestGuardianshipManager_RemoveGuardian_FindError(t *testing.T) {
	guardRepo := &stubGuardianshipRepo{findErr: errors.New("db error")}
	manager := NewManagerService(guardRepo, &stubChildDomainRepo{}, &stubUserDomainRepo{})

	removed, err := manager.RemoveGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1))

	require.Error(t, err)
	assert.Nil(t, removed)
	assert.Contains(t, fmt.Sprintf("%-v", err), "find guardians failed")
}

// seqGuardRepo 提供按调用序列返回不同结果的 FindByChildID，帮助并发测试
type seqGuardRepo struct {
	mu        sync.Mutex
	calls     int
	responses [][]*Guardianship
}

func (s *seqGuardRepo) Create(context.Context, *Guardianship) error                { return nil }
func (s *seqGuardRepo) FindByID(context.Context, idutil.ID) (*Guardianship, error) { return nil, nil }
func (s *seqGuardRepo) FindByChildID(ctx context.Context, id meta.ID) ([]*Guardianship, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls < len(s.responses) {
		res := s.responses[s.calls]
		s.calls++
		return res, nil
	}
	return s.responses[len(s.responses)-1], nil
}
func (s *seqGuardRepo) FindByUserID(context.Context, meta.ID) ([]*Guardianship, error) {
	return nil, nil
}
func (s *seqGuardRepo) FindByUserIDAndChildID(context.Context, meta.ID, meta.ID) (*Guardianship, error) {
	return nil, nil
}
func (s *seqGuardRepo) IsGuardian(context.Context, meta.ID, meta.ID) (bool, error) { return false, nil }
func (s *seqGuardRepo) Update(context.Context, *Guardianship) error                { return nil }

func TestGuardianshipManager_AddGuardian_ConcurrentDuplicateDetection(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.FromUint64(1)}}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.FromUint64(2)}}

	existing := &Guardianship{User: meta.FromUint64(2), Child: meta.FromUint64(1)}
	seq := &seqGuardRepo{
		responses: [][]*Guardianship{
			{},         // first caller sees none
			{existing}, // second caller sees existing guardian
		},
	}

	manager := NewManagerService(seq, childRepo, userRepo)

	var wg sync.WaitGroup
	wg.Add(2)

	startCh := make(chan struct{})
	results := make([]struct {
		g   *Guardianship
		err error
	}, 2)

	for i := 0; i < 2; i++ {
		idx := i
		go func() {
			defer wg.Done()
			<-startCh
			g, err := manager.AddGuardian(context.Background(), meta.FromUint64(2), meta.FromUint64(1), RelParent)
			results[idx].g = g
			results[idx].err = err
		}()
	}

	// 同时开始，两者几乎同时调用 FindByChildID
	close(startCh)
	wg.Wait()

	// 期望：一个成功，另一个因为已存在而失败
	var success, duplicated int
	for _, r := range results {
		if r.err == nil && r.g != nil {
			success++
		} else if r.err != nil {
			if contains(fmt.Sprintf("%-v", r.err), "guardian already exists") {
				duplicated++
			}
		}
	}

	// 要求至少有一个成功和至少一个重复错误
	assert.GreaterOrEqual(t, success, 1)
	assert.GreaterOrEqual(t, duplicated, 1)
}

// contains 方便在断言中检查子串
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (indexOf(s, sub) >= 0))
}

// indexOf 是一个简单实现，避免引入 strings 包以保持测试文件与其余代码风格一致
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
