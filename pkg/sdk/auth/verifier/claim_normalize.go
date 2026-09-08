package verifier

import "strings"

func applyOrg(claims *TokenClaims, orgRaw string) {
	if claims != nil {
		claims.OrgID = strings.TrimSpace(orgRaw)
	}
}
