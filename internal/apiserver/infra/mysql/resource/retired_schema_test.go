package resource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapperRejectsUnmigratedResourceSchema(t *testing.T) {
	_, err := NewMapper().ToBO(&ResourcePO{Key: "qs:evaluation:collection:assessments", Actions: `["retry"]`, AttributeSchema: `{"version":1,"attributes":[{"key":"object.origin_type","type":"string"}]}`})
	require.Error(t, err)
}
