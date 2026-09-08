package authorization

import (
	"context"
	"strconv"

	authorizationapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/authorization"
	appquery "github.com/FangcunMount/iam/v5/internal/apiserver/application/suggest/queryprofile"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/suggest/visibility"
)

// FactsReader 从 AuthZ 查询 Suggest 授权事实。
type FactsReader struct {
	permissions authorizationapp.RoutePermissionChecker
}

// NewFactsReader 创建 reader。
func NewFactsReader(permissions authorizationapp.RoutePermissionChecker) *FactsReader {
	return &FactsReader{permissions: permissions}
}

// ReadAuthorizationFacts 实现 queryprofile.AuthorizationFactsReader。
func (r *FactsReader) ReadAuthorizationFacts(
	ctx context.Context,
	principal visibility.Principal,
) (visibility.AuthorizationFacts, error) {
	if r == nil || r.permissions == nil {
		return visibility.AuthorizationFacts{}, nil
	}
	sub := "user:" + strconv.FormatInt(principal.OperatorID, 10)

	listAllowed, err := r.permissions.CheckRoutePermission(
		ctx, sub, appquery.ResourceIAMProfileCollection, "list_all",
	)
	if err != nil {
		return visibility.AuthorizationFacts{}, err
	}
	if listAllowed {
		mobileAllowed, err := r.permissions.CheckRoutePermission(
			ctx, sub, appquery.ResourceIAMProfileCollection, "search_by_mobile_all",
		)
		if err != nil {
			return visibility.AuthorizationFacts{}, err
		}
		return visibility.AuthorizationFacts{
			AllProfilesAllowed:             true,
			AllProfilesMobileSearchAllowed: mobileAllowed,
		}, nil
	}

	mobileOK, err := r.permissions.CheckRoutePermission(
		ctx, sub, appquery.ResourceIAMProfileCollection, appquery.ActionSearchByMobile,
	)
	if err != nil {
		return visibility.AuthorizationFacts{}, err
	}
	return visibility.AuthorizationFacts{ScopedMobileSearchAllowed: mobileOK}, nil
}

var _ appquery.AuthorizationFactsReader = (*FactsReader)(nil)
