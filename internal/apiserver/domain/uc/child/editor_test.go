package child

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

type stubChildRepository struct {
	child      *Child
	findErr    error
	findCalls  int
	updateArgs []*Child
}

func (s *stubChildRepository) Create(context.Context, *Child) error { return nil }
func (s *stubChildRepository) FindByID(ctx context.Context, id meta.ID) (*Child, error) {
	s.findCalls++
	if s.findErr != nil {
		return nil, s.findErr
	}
	if s.child == nil {
		return nil, fmt.Errorf("child %d not found", id.ToUint64())
	}
	return s.child, nil
}
func (s *stubChildRepository) FindByName(context.Context, string) (*Child, error) {
	return nil, s.findErr
}
func (s *stubChildRepository) FindByIDCard(context.Context, meta.IDCard) (*Child, error) {
	return nil, s.findErr
}
func (s *stubChildRepository) FindListByName(context.Context, string) ([]*Child, error) {
	return nil, s.findErr
}
func (s *stubChildRepository) FindListByNameAndBirthday(context.Context, string, meta.Birthday) ([]*Child, error) {
	return nil, s.findErr
}
func (s *stubChildRepository) FindSimilar(context.Context, string, meta.Gender, meta.Birthday) ([]*Child, error) {
	return nil, s.findErr
}
func (s *stubChildRepository) Update(ctx context.Context, child *Child) error {
	s.updateArgs = append(s.updateArgs, child)
	return s.findErr
}

type stubChildValidator struct {
	renameErr        error
	updateProfileErr error
}

func (s *stubChildValidator) ValidateRegister(context.Context, string, meta.Gender, meta.Birthday) error {
	return nil
}
func (s *stubChildValidator) ValidateRename(string) error {
	return s.renameErr
}
func (s *stubChildValidator) ValidateUpdateProfile(meta.Gender, meta.Birthday) error {
	return s.updateProfileErr
}

func TestChildProfileEditor_RenameSuccess(t *testing.T) {
	child := &Child{ID: meta.NewID(1), Name: "Old"}
	repo := &stubChildRepository{child: child}
	editor := NewProfileService(repo, &stubChildValidator{})

	updated, err := editor.Rename(context.Background(), child.ID, "NewName")

	require.NoError(t, err)
	assert.Equal(t, "NewName", child.Name)
	assert.Same(t, child, updated)
	assert.Equal(t, 1, repo.findCalls)
}

func TestChildProfileEditor_RenameValidatorError(t *testing.T) {
	repo := &stubChildRepository{child: &Child{ID: meta.NewID(1), Name: "Old"}}
	editor := NewProfileService(repo, &stubChildValidator{renameErr: errors.New("invalid name")})

	updated, err := editor.Rename(context.Background(), repo.child.ID, "bad")

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, 0, repo.findCalls, "repository should not be called when validation fails")
}

func TestChildProfileEditor_RenameRepoError(t *testing.T) {
	repo := &stubChildRepository{child: &Child{ID: meta.NewID(1)}, findErr: errors.New("db error")}
	editor := NewProfileService(repo, &stubChildValidator{})

	updated, err := editor.Rename(context.Background(), repo.child.ID, "Name")

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, 1, repo.findCalls)
}

func TestChildProfileEditor_UpdateProfileSuccess(t *testing.T) {
	child := &Child{ID: meta.NewID(2)}
	repo := &stubChildRepository{child: child}
	editor := NewProfileService(repo, &stubChildValidator{})

	birthday := meta.NewBirthday("2020-05-06")
	updated, err := editor.UpdateProfile(context.Background(), child.ID, meta.GenderFemale, birthday)

	require.NoError(t, err)
	assert.Same(t, child, updated)
	assert.Equal(t, meta.GenderFemale, child.Gender)
	assert.True(t, child.Birthday.Equal(birthday))
}

func TestChildProfileEditor_UpdateProfileValidatorError(t *testing.T) {
	repo := &stubChildRepository{child: &Child{ID: meta.NewID(3)}}
	editor := NewProfileService(repo, &stubChildValidator{updateProfileErr: errors.New("bad birthday")})

	updated, err := editor.UpdateProfile(context.Background(), repo.child.ID, meta.GenderMale, meta.Birthday{})

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, 0, repo.findCalls)
}

func TestChildProfileEditor_UpdateHeightWeight(t *testing.T) {
	child := &Child{ID: meta.NewID(4)}
	repo := &stubChildRepository{child: child}
	editor := NewProfileService(repo, &stubChildValidator{})

	height, err := meta.NewHeightFromFloat(150.4)
	require.NoError(t, err)
	weight, err := meta.NewWeightFromFloat(45.1)
	require.NoError(t, err)

	updated, err := editor.UpdateHeightWeight(context.Background(), child.ID, height, weight)

	require.NoError(t, err)
	assert.Same(t, child, updated)
	assert.Equal(t, height.Tenths(), child.Height.Tenths())
	assert.Equal(t, weight.Tenths(), child.Weight.Tenths())
}

func TestChildProfileEditor_UpdateIDCard(t *testing.T) {
	child := &Child{ID: meta.NewID(5)}
	repo := &stubChildRepository{child: child}
	editor := NewProfileService(repo, &stubChildValidator{})

	idCard := meta.NewIDCard("tester", "111")
	updated, err := editor.UpdateIDCard(context.Background(), child.ID, idCard)

	require.NoError(t, err)
	assert.Same(t, child, updated)
	assert.True(t, child.IDCard.Equal(idCard))
}
