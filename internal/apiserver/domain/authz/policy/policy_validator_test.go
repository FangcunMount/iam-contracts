package policy_test

import (
	"context"
	"testing"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

// role repo stub
type roleRepoStub struct {
	found *role.Role
	err   error
}

func (r *roleRepoStub) Create(ctx context.Context, rn *role.Role) error { return nil }
func (r *roleRepoStub) Update(ctx context.Context, rn *role.Role) error { return nil }
func (r *roleRepoStub) Delete(ctx context.Context, id meta.ID) error    { return nil }
func (r *roleRepoStub) FindByID(ctx context.Context, id meta.ID) (*role.Role, error) {
	return r.found, r.err
}
func (r *roleRepoStub) FindByName(ctx context.Context, tenantID, name string) (*role.Role, error) {
	return nil, nil
}
func (r *roleRepoStub) List(ctx context.Context, tenantID string, offset, limit int) ([]*role.Role, int64, error) {
	return nil, 0, nil
}

// resource repo stub
type resRepoStub struct {
	found  *resource.Resource
	valid  bool
	err    error
	valErr error
}

func (r *resRepoStub) Create(ctx context.Context, res *resource.Resource) error { return nil }
func (r *resRepoStub) Update(ctx context.Context, res *resource.Resource) error { return nil }
func (r *resRepoStub) Delete(ctx context.Context, id resource.ResourceID) error { return nil }
func (r *resRepoStub) FindByID(ctx context.Context, id resource.ResourceID) (*resource.Resource, error) {
	return r.found, r.err
}
func (r *resRepoStub) FindByKey(ctx context.Context, key string) (*resource.Resource, error) {
	return nil, nil
}
func (r *resRepoStub) List(ctx context.Context, offset, limit int) ([]*resource.Resource, int64, error) {
	return nil, 0, nil
}
func (r *resRepoStub) ValidateAction(ctx context.Context, resourceKey, action string) (bool, error) {
	return r.valid, r.valErr
}

func TestValidateAddAndRoleResourceChecks(t *testing.T) {
	v := policy.NewValidator(&roleRepoStub{}, &resRepoStub{})

	// missing params
	err := v.ValidateAddPolicyParameters(0, resource.NewResourceID(1), "act", "t", "by")
	require.Error(t, err)
	var zero resource.ResourceID
	err = v.ValidateAddPolicyParameters(1, zero, "act", "t", "by")
	require.Error(t, err)
	err = v.ValidateAddPolicyParameters(1, resource.NewResourceID(1), "", "t", "by")
	require.Error(t, err)
	err = v.ValidateAddPolicyParameters(1, resource.NewResourceID(1), "a", "", "by")
	require.Error(t, err)
	err = v.ValidateAddPolicyParameters(1, resource.NewResourceID(1), "a", "t", "")
	require.Error(t, err)

	// CheckRoleExistsAndTenant: role not found
	rr := &roleRepoStub{found: nil, err: errors.WithCode(code.ErrRoleNotFound, "nf")}
	v2 := policy.NewValidator(rr, &resRepoStub{})
	_, err = v2.CheckRoleExistsAndTenant(context.Background(), 123, "t")
	require.Error(t, err)
	require.True(t, errors.IsCode(err, code.ErrRoleNotFound))

	// role exists but tenant mismatch
	rrole := &role.Role{ID: meta.FromUint64(10), Name: "r", TenantID: "A"}
	rr2 := &roleRepoStub{found: rrole, err: nil}
	v3 := policy.NewValidator(rr2, &resRepoStub{})
	_, err = v3.CheckRoleExistsAndTenant(context.Background(), 10, "B")
	require.Error(t, err)
	require.True(t, errors.IsCode(err, code.ErrPermissionDenied))

	// ok
	_, err = v3.CheckRoleExistsAndTenant(context.Background(), 10, "A")
	require.NoError(t, err)

	// CheckResourceExistsAndValidateAction: resource not found
	rs := &resRepoStub{found: nil, err: errors.WithCode(code.ErrResourceNotFound, "nf")}
	v4 := policy.NewValidator(rr2, rs)
	_, err = v4.CheckResourceExistsAndValidateAction(context.Background(), resource.NewResourceID(1), "act")
	require.Error(t, err)
	require.True(t, errors.IsCode(err, code.ErrResourceNotFound))

	// resource found but validate action error
	rs2 := &resRepoStub{found: &resource.Resource{ID: resource.NewResourceID(2), Key: "k"}, valid: false, valErr: errors.New("boom")}
	v5 := policy.NewValidator(rr2, rs2)
	_, err = v5.CheckResourceExistsAndValidateAction(context.Background(), resource.NewResourceID(2), "act")
	require.Error(t, err)

	// resource found but invalid action
	rs3 := &resRepoStub{found: &resource.Resource{ID: resource.NewResourceID(3), Key: "k3"}, valid: false, valErr: nil}
	v6 := policy.NewValidator(rr2, rs3)
	_, err = v6.CheckResourceExistsAndValidateAction(context.Background(), resource.NewResourceID(3), "act")
	require.Error(t, err)
	require.True(t, errors.IsCode(err, code.ErrInvalidAction))

	// ok
	rs4 := &resRepoStub{found: &resource.Resource{ID: resource.NewResourceID(4), Key: "k4"}, valid: true, valErr: nil}
	v7 := policy.NewValidator(rr2, rs4)
	key, err := v7.CheckResourceExistsAndValidateAction(context.Background(), resource.NewResourceID(4), "act")
	require.NoError(t, err)
	require.Equal(t, "k4", key)
}
