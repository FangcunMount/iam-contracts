package resource_test

import (
	"context"
	"testing"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/stretchr/testify/require"
)

// repo stub
type repoStub struct {
	found *resource.Resource
	err   error
}

func (r *repoStub) Create(ctx context.Context, res *resource.Resource) error { return nil }
func (r *repoStub) Update(ctx context.Context, res *resource.Resource) error { return nil }
func (r *repoStub) Delete(ctx context.Context, id resource.ResourceID) error { return nil }
func (r *repoStub) FindByID(ctx context.Context, id resource.ResourceID) (*resource.Resource, error) {
	return r.found, r.err
}
func (r *repoStub) FindByKey(ctx context.Context, key string) (*resource.Resource, error) {
	return r.found, r.err
}
func (r *repoStub) List(ctx context.Context, offset, limit int) ([]*resource.Resource, int64, error) {
	return nil, 0, nil
}
func (r *repoStub) ValidateAction(ctx context.Context, resourceKey, action string) (bool, error) {
	return false, nil
}

func TestNewResourceAndHasAction(t *testing.T) {
	r := resource.NewResource("app:dom:typ:*", []string{"read", "write"}, resource.WithDisplayName("X"))
	require.Equal(t, "app:dom:typ:*", r.Key)
	require.True(t, r.HasAction("read"))
	require.False(t, r.HasAction("delete"))
}

func TestValidator_CheckKeyUnique_And_Params(t *testing.T) {
	v := resource.NewValidator(&repoStub{})

	// empty key
	err := v.CheckKeyUnique(context.Background(), "")
	require.Error(t, err)
	require.True(t, errors.IsCode(err, code.ErrInvalidArgument))

	// repo says exists
	rs := &repoStub{found: &resource.Resource{Key: "k1"}}
	v2 := resource.NewValidator(rs)
	err = v2.CheckKeyUnique(context.Background(), "k1")
	require.Error(t, err)
	require.True(t, errors.IsCode(err, code.ErrResourceAlreadyExists))

	// repo returns NotFound -> ok
	rs2 := &repoStub{found: nil, err: errors.WithCode(code.ErrResourceNotFound, "not found")}
	v3 := resource.NewValidator(rs2)
	err = v3.CheckKeyUnique(context.Background(), "k2")
	require.NoError(t, err)

	// repo returns unexpected error -> wrapped
	rs3 := &repoStub{found: nil, err: errors.New("db fail")}
	v4 := resource.NewValidator(rs3)
	err = v4.CheckKeyUnique(context.Background(), "k3")
	require.Error(t, err)
}

func TestValidator_ValidateCreateAndUpdateParameters(t *testing.T) {
	v := resource.NewValidator(&repoStub{})
	// missing fields
	err := v.ValidateCreateParameters("", "dn", "app", "dom", "typ", []string{"a"})
	require.Error(t, err)

	err = v.ValidateCreateParameters("k", "", "app", "dom", "typ", []string{"a"})
	require.Error(t, err)

	err = v.ValidateCreateParameters("k", "dn", "", "dom", "typ", []string{"a"})
	require.Error(t, err)

	err = v.ValidateCreateParameters("k", "dn", "app", "", "typ", []string{"a"})
	require.Error(t, err)

	err = v.ValidateCreateParameters("k", "dn", "app", "dom", "", []string{"a"})
	require.Error(t, err)

	// actions empty
	err = v.ValidateCreateParameters("k", "dn", "app", "dom", "typ", []string{})
	require.Error(t, err)

	// update with nil actions -> ok
	err = v.ValidateUpdateParameters(nil)
	require.NoError(t, err)
	// update with empty actions -> error
	err = v.ValidateUpdateParameters([]string{})
	require.Error(t, err)
}

func TestValidator_CheckResourceExists(t *testing.T) {
	// id zero
	v := resource.NewValidator(&repoStub{})
	var zero resource.ResourceID
	_, err := v.CheckResourceExists(context.Background(), zero)
	require.Error(t, err)

	// repo returns not found
	rs := &repoStub{found: nil, err: errors.WithCode(code.ErrResourceNotFound, "not found")}
	v2 := resource.NewValidator(rs)
	_, err = v2.CheckResourceExists(context.Background(), resource.NewResourceID(1))
	require.Error(t, err)
	require.True(t, errors.IsCode(err, code.ErrResourceNotFound))

	// repo returns other err
	rs2 := &repoStub{found: nil, err: errors.New("boom")}
	v3 := resource.NewValidator(rs2)
	_, err = v3.CheckResourceExists(context.Background(), resource.NewResourceID(1))
	require.Error(t, err)

	// repo returns found
	res := &resource.Resource{ID: resource.NewResourceID(2), Key: "k"}
	rs3 := &repoStub{found: res, err: nil}
	v4 := resource.NewValidator(rs3)
	got, err := v4.CheckResourceExists(context.Background(), resource.NewResourceID(2))
	require.NoError(t, err)
	require.Equal(t, "k", got.Key)
}
