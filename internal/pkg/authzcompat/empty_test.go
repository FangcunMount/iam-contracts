package authzcompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetiredFieldsRejectNonemptyAndMalformedInputs(t *testing.T) {
	for _, field := range []string{"all_of", "attributes"} {
		for _, raw := range []string{"", `null`, `{}`, `{"version":1}`, `{"version":1,"` + field + `":[]}`} {
			require.NoError(t, ValidateEmpty([]byte(raw), field), raw)
		}
		for _, raw := range []string{`[]`, `true`, `{"version":2}`, `{"version":0}`, `{"version":null}`, `{"unknown":[]}`, `{"` + field + `":[{}]}`, `{"` + field + `":""}`, `{} {}`, `{"version":2,"version":1}`, `{"` + field + `":[{}],"` + field + `":[]}`} {
			require.Error(t, ValidateEmpty([]byte(raw), field), raw)
		}
	}
	var c Constraints
	require.Error(t, json.Unmarshal([]byte(`{"all_of":[{}]}`), &c))
	var s Schema
	require.Error(t, json.Unmarshal([]byte(`{"attributes":[{}]}`), &s))
}
