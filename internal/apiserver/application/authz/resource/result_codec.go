package resource

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type resourceResult struct {
	ID          string   `json:"id"`
	Key         string   `json:"key"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	AppName     string   `json:"app_name"`
	Domain      string   `json:"domain"`
	Type        string   `json:"type"`
	Actions     []string `json:"actions"`
}

func resourceResultCodec(current func() *domain.Resource, restore func(domain.Resource)) *operation.ResultCodec {
	return operation.ProjectResult("iam.resource-result/v1", func() (resourceResult, error) {
		value := current()
		if value == nil || value.ID.Uint64() == 0 {
			return resourceResult{}, operation.ErrInvalid
		}
		return resourceResult{ID: value.ID.String(), Key: value.KeyString(), DisplayName: value.DisplayName, Description: value.Description, AppName: value.AppName, Domain: value.Domain, Type: value.Type, Actions: value.ActionStrings()}, nil
	}, func(value resourceResult) error {
		id, err := meta.ParseID(value.ID)
		if err != nil || id.IsZero() || id.String() != value.ID {
			return operation.ErrInvalid
		}
		resource, err := domain.RestoreResource(value.Key, value.Actions, domain.WithID(domain.NewResourceID(id.Uint64())), domain.WithDisplayName(value.DisplayName), domain.WithDescription(value.Description), domain.WithAppName(value.AppName), domain.WithDomain(value.Domain), domain.WithType(value.Type))
		if err != nil {
			return err
		}
		restore(resource)
		return nil
	})
}
