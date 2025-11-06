package guardianship

import (
	"context"
	"errors"
	"fmt"
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
	return s.childrenResults[id.ToUint64()], nil
}
func (s *stubGuardianshipRepo) FindByUserID(context.Context, meta.ID) ([]*Guardianship, error) {
	return nil, nil
}
func (s *stubGuardianshipRepo) FindByUserIDAndChildID(context.Context, meta.ID, meta.ID) (*Guardianship, error) {
	return nil, nil
}
func (s *stubGuardianshipRepo) IsGuardian(context.Context, meta.ID, meta.ID) (bool, error) { return false, nil }
func (s *stubGuardianshipRepo) Update(context.Context, *Guardianship) error               { return nil }

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

type idutilID interface{ Uint64() uint64 }

func TestGuardianshipManager_AddGuardianSuccess(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.NewID(1)}}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.NewID(2)}}
	guardRepo := &stubGuardianshipRepo{childrenResults: make(map[uint64][]*Guardianship)}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.NewID(2), meta.NewID(1), RelParent)

	require.NoError(t, err)
	require.NotNil(t, guardian)
	assert.Equal(t, meta.NewID(2), guardian.User)
	assert.Equal(t, meta.NewID(1), guardian.Child)
	assert.Equal(t, RelParent, guardian.Rel)
	assert.False(t, guardian.EstablishedAt.IsZero())
}

func TestGuardianshipManager_AddGuardian_Duplicate(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.NewID(1)}}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.NewID(2)}}
	existing := &Guardianship{User: meta.NewID(2), Child: meta.NewID(1)}
	guardRepo := &stubGuardianshipRepo{
		childrenResults: map[uint64][]*Guardianship{
			1: {existing},
		},
	}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.NewID(2), meta.NewID(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "guardian already exists")
}

func TestGuardianshipManager_AddGuardian_ChildNotFound(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: nil}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.NewID(2)}}
	guardRepo := &stubGuardianshipRepo{}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.NewID(2), meta.NewID(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "child not found")
}

func TestGuardianshipManager_AddGuardian_UserRepoError(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.NewID(1)}}
	userRepo := &stubUserDomainRepo{err: errors.New("db error")}
	guardRepo := &stubGuardianshipRepo{}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.NewID(2), meta.NewID(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "find user failed")
}

func TestGuardianshipManager_AddGuardian_FindByChildError(t *testing.T) {
	childRepo := &stubChildDomainRepo{child: &childdomain.Child{ID: meta.NewID(1)}}
	userRepo := &stubUserDomainRepo{user: &userdomain.User{ID: meta.NewID(2)}}
	guardRepo := &stubGuardianshipRepo{findErr: errors.New("db error")}

	manager := NewManagerService(guardRepo, childRepo, userRepo)

	guardian, err := manager.AddGuardian(context.Background(), meta.NewID(2), meta.NewID(1), RelParent)

	require.Error(t, err)
	assert.Nil(t, guardian)
	assert.Contains(t, fmt.Sprintf("%-v", err), "find guardians failed")
}

func TestGuardianshipManager_RemoveGuardianSuccess(t *testing.T) {
	target := &Guardianship{User: meta.NewID(2), Child: meta.NewID(1)}
	guardRepo := &stubGuardianshipRepo{
		childrenResults: map[uint64][]*Guardianship{
			1: {target},
		},
	}
	manager := NewManagerService(guardRepo, &stubChildDomainRepo{}, &stubUserDomainRepo{})

	removed, err := manager.RemoveGuardian(context.Background(), meta.NewID(2), meta.NewID(1))

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

	removed, err := manager.RemoveGuardian(context.Background(), meta.NewID(2), meta.NewID(1))

	require.Error(t, err)
	assert.Nil(t, removed)
	assert.Contains(t, fmt.Sprintf("%-v", err), "active guardian not found")
}
