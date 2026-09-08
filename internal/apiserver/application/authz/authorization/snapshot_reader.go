package authorization

import (
	"context"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// SnapshotRuntime supplies the active subject authorization projection.
type SnapshotRuntime interface {
	GetAuthorizationSnapshot(context.Context, subject.Ref, string) (SubjectSnapshot, error)
}

// SnapshotReader exposes the application query for one subject's current
// direct roles, effective roles, permissions, and policy version.
type SnapshotReader struct {
	runtime SnapshotRuntime
}

func NewSnapshotReader(runtime SnapshotRuntime) *SnapshotReader {
	return &SnapshotReader{runtime: runtime}
}

func (r *SnapshotReader) Read(ctx context.Context, sub subject.Ref, appName string) (SubjectSnapshot, error) {
	if r == nil || r.runtime == nil {
		return SubjectSnapshot{}, perrors.WithCode(code.ErrInternalServerError, "authorization runtime is unavailable")
	}
	if sub.IsZero() || strings.TrimSpace(appName) == "" {
		return SubjectSnapshot{}, perrors.WithCode(code.ErrInvalidArgument, "主体和应用名称必填")
	}
	return r.runtime.GetAuthorizationSnapshot(ctx, sub, appName)
}
