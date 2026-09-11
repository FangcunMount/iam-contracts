package runtime_test

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	authzruntime "github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/runtime"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRejectsSensitiveGrantForStandardRole(t *testing.T) {
	dataset := assessmentDataset(t)
	dataset.Grants = nil
	grant, err := permissiongrant.NewSystem(meta.ID(11), resource.ResourceID{}, "iam:*:*:*", "*", "seed")
	require.NoError(t, err)
	dataset.Grants = append(dataset.Grants, &grant)
	_, err = authzruntime.BuildSnapshot(dataset, time.Now())
	require.Error(t, err)
	dataset.Roles[0].ManagementProtection = role.ManagementProtected
	_, err = authzruntime.BuildSnapshot(dataset, time.Now())
	require.NoError(t, err)
}
