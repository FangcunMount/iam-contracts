package assignment

import (
	"encoding/json"
	"slices"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/scope"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type assignmentScopeResult struct {
	OrgID    string   `json:"org_id"`
	Kind     string   `json:"kind"`
	StoreIDs []string `json:"store_ids"`
}
type assignmentResult struct {
	ID          string                 `json:"id"`
	SubjectType string                 `json:"subject_type"`
	SubjectID   string                 `json:"subject_id"`
	RoleID      string                 `json:"role_id"`
	GrantedBy   string                 `json:"granted_by"`
	Scope       *assignmentScopeResult `json:"scope,omitempty"`
}

func assignmentResultCodec(result *grantResult) *operation.ResultCodec {
	return operation.ProjectResult("iam.assignment-result/v1", func() (assignmentResult, error) {
		value := result.Assignment
		if value == nil || value.ID.Uint64() == 0 {
			return assignmentResult{}, operation.ErrInvalid
		}
		snapshot := assignmentResult{ID: value.ID.String(), SubjectType: value.SubjectTypeString(), SubjectID: value.SubjectID.String(), RoleID: value.RoleID.String(), GrantedBy: value.GrantedBy}
		if data, ok := value.Scope(); ok {
			stores := make([]string, 0, len(data.StoreIDs()))
			for _, id := range data.StoreIDs() {
				stores = append(stores, id.String())
			}
			snapshot.Scope = &assignmentScopeResult{OrgID: data.OrgID().String(), Kind: string(data.Kind()), StoreIDs: stores}
		}
		return snapshot, nil
	}, func(value assignmentResult) error {
		parse := func(raw string) (meta.ID, error) {
			id, err := meta.ParseID(raw)
			if err != nil || id.IsZero() || id.String() != raw {
				return 0, operation.ErrInvalid
			}
			return id, nil
		}
		id, err := parse(value.ID)
		if err != nil {
			return err
		}
		subjectID, err := parse(value.SubjectID)
		if err != nil {
			return err
		}
		roleID, err := parse(value.RoleID)
		if err != nil {
			return err
		}
		options := []domain.Option{domain.WithID(domain.NewAssignmentID(id.Uint64())), domain.WithGrantedBy(value.GrantedBy)}
		if value.Scope != nil {
			org, err := parse(value.Scope.OrgID)
			if err != nil {
				return err
			}
			stores := make([]meta.ID, 0, len(value.Scope.StoreIDs))
			for _, raw := range value.Scope.StoreIDs {
				id, err := parse(raw)
				if err != nil {
					return err
				}
				stores = append(stores, id)
			}
			data, err := scope.New(org, scope.Kind(value.Scope.Kind), stores)
			if err != nil {
				return err
			}
			if !slices.Equal(stores, data.StoreIDs()) {
				return operation.ErrInvalid
			}
			options = append(options, domain.WithScope(data))
		}
		assignment, err := domain.NewAssignment(domain.SubjectType(value.SubjectType), subjectID, roleID, options...)
		if err != nil {
			return err
		}
		result.Assignment = &assignment
		return nil
	})
}

type assignmentWriteResult struct {
	PolicyVersion int64           `json:"policy_version"`
	Changed       bool            `json:"changed"`
	Mutation      json.RawMessage `json:"mutation"`
}

func versionedAssignmentResult(inner *operation.ResultCodec, version *int64, changed *bool) *operation.ResultCodec {
	if inner == nil {
		inner = operation.EmptyResultCodec()
	}
	return operation.ProjectResult("iam.assignment-write-result/v1", func() (assignmentWriteResult, error) {
		if *version < 1 {
			return assignmentWriteResult{}, operation.ErrInvalid
		}
		body, err := inner.Capture()
		if err != nil {
			return assignmentWriteResult{}, err
		}
		return assignmentWriteResult{PolicyVersion: *version, Changed: *changed, Mutation: body}, nil
	}, func(value assignmentWriteResult) error {
		if value.PolicyVersion < 1 {
			return operation.ErrInvalid
		}
		if err := inner.Restore(value.Mutation); err != nil {
			return err
		}
		*version = value.PolicyVersion
		*changed = value.Changed
		return nil
	})
}

type replacementResult struct {
	DirectRoles   []string `json:"direct_roles"`
	PolicyVersion int64    `json:"policy_version"`
	Changed       bool     `json:"changed"`
}

func replacementResultCodec(result *ReplaceManagedAssignmentsResult) *operation.ResultCodec {
	return operation.ProjectResult("iam.managed-assignments-result/v1", func() (replacementResult, error) {
		if result.PolicyVersion < 1 {
			return replacementResult{}, operation.ErrInvalid
		}
		return replacementResult{DirectRoles: slices.Clone(result.DirectRoles), PolicyVersion: result.PolicyVersion, Changed: result.Changed}, nil
	}, func(value replacementResult) error {
		if value.PolicyVersion < 1 || !slices.IsSorted(value.DirectRoles) {
			return operation.ErrInvalid
		}
		*result = ReplaceManagedAssignmentsResult{DirectRoles: slices.Clone(value.DirectRoles), PolicyVersion: value.PolicyVersion, Changed: value.Changed}
		return nil
	})
}
