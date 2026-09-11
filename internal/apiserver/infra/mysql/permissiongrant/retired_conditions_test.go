package permissiongrant

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapperRejectsConditionalRowsBeforeDomainRestore(t *testing.T) {
	for _, raw := range []string{`{"version":1,"all_of":[{"key":"object.origin_type"}]}`, `{"version":99,"all_of":[]}`, `broken`} {
		_, err := (Mapper{}).ToBO(&GrantPO{ConstraintSet: raw})
		require.Error(t, err)
	}
}
