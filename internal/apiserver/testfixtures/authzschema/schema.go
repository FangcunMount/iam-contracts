// Package authzschema provides business independent authorization examples.
package authzschema

import "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/attribute"

const AttributeKey = "object.status"

func Schema() attribute.Schema {
	s, err := attribute.NewSchema([]attribute.Definition{{Key: AttributeKey, Type: attribute.TypeString, AllowedStringValues: []string{"active", "paused"}}})
	if err != nil {
		panic(err)
	}
	return s
}
