package grant

import (
	"encoding/json"
	"fmt"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	rp "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/attribute"
)

func ResourceFromPO(p *rp.ResourcePO) (*resource.Resource, error) {
	copy := *p
	copy.AttributeSchema = ""
	return rp.NewMapper().ToBO(&copy)
}
func (g Grant) ValidateAgainst(r resource.Resource, raw string) error {
	if g.ResourceID.Uint64() != r.ID.Uint64() || g.ResourceKeyString() != r.KeyString() || !r.HasAction(g.ActionString()) {
		return fmt.Errorf("historical grant catalog mismatch")
	}
	var schema attribute.Schema
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &schema); err != nil {
			return err
		}
	}
	return g.Constraints.ValidateAgainst(schema)
}
