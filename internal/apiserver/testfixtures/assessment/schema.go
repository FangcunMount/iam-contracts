// Package assessment contains the QS contract used only by integration tests.
package assessment

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/attribute"
)

const AttributeKey = "object.origin_type"
const Resource = "qs:evaluation:collection:assessments"
const Service = "qs-apiserver.svc"

func Schema() attribute.Schema {
	s, err := attribute.NewSchema([]attribute.Definition{{Key: AttributeKey, Type: attribute.TypeString, AllowedStringValues: []string{"adhoc", "plan"}}})
	if err != nil {
		panic(err)
	}
	return s
}
