package verifier

import "github.com/lestrrat-go/jwx/v2/jwa"

func (s *LocalVerifyStrategy) getAllowedAlgorithms() []jwa.SignatureAlgorithm {
	if s.config == nil || len(s.config.Algorithms) == 0 {
		return []jwa.SignatureAlgorithm{jwa.RS256}
	}

	algorithms := make([]jwa.SignatureAlgorithm, 0, len(s.config.Algorithms))
	for _, alg := range s.config.Algorithms {
		switch alg {
		case "RS256":
			algorithms = append(algorithms, jwa.RS256)
		case "RS384":
			algorithms = append(algorithms, jwa.RS384)
		case "RS512":
			algorithms = append(algorithms, jwa.RS512)
		case "ES256":
			algorithms = append(algorithms, jwa.ES256)
		case "ES384":
			algorithms = append(algorithms, jwa.ES384)
		case "ES512":
			algorithms = append(algorithms, jwa.ES512)
		case "PS256":
			algorithms = append(algorithms, jwa.PS256)
		case "PS384":
			algorithms = append(algorithms, jwa.PS384)
		case "PS512":
			algorithms = append(algorithms, jwa.PS512)
		case "EdDSA":
			algorithms = append(algorithms, jwa.EdDSA)
		}
	}

	if len(algorithms) == 0 {
		return []jwa.SignatureAlgorithm{jwa.RS256}
	}
	return algorithms
}
