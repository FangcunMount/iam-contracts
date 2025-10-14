package account

import (
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

func accountIDString(id domain.AccountID) string {
	return idutil.ID(id).String()
}
