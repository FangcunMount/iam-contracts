package jwks

import "github.com/lestrrat-go/jwx/v2/jwk"

func parseJWKSResponse(payload []byte) (jwk.Set, error) {
	return jwk.Parse(payload)
}
