package verifier

import authnv3 "github.com/FangcunMount/iam/v5/api/grpc/iam/authn/v3"

func buildVerifyMetadataFromProto(metadata *authnv3.TokenMetadata) *VerifyMetadata {
	if metadata == nil {
		return nil
	}
	result := &VerifyMetadata{
		TokenType: metadata.GetTokenType(),
		Status:    metadata.GetStatus(),
	}
	if metadata.GetIssuedAt() != nil {
		result.IssuedAt = metadata.GetIssuedAt().AsTime()
	}
	if metadata.GetExpiresAt() != nil {
		result.ExpiresAt = metadata.GetExpiresAt().AsTime()
	}
	return result
}

func buildVerifyMetadataFromClaims(claims *TokenClaims) *VerifyMetadata {
	if claims == nil {
		return nil
	}
	return &VerifyMetadata{
		TokenType: tokenTypeToProto(claims.TokenType),
		Status:    authnv3.TokenStatus_TOKEN_STATUS_VALID,
		IssuedAt:  claims.IssuedAt,
		ExpiresAt: claims.ExpiresAt,
	}
}

func tokenTypeToProto(tokenType string) authnv3.TokenType {
	switch tokenType {
	case "refresh":
		return authnv3.TokenType_TOKEN_TYPE_REFRESH
	case "", "access":
		return authnv3.TokenType_TOKEN_TYPE_ACCESS
	default:
		return authnv3.TokenType_TOKEN_TYPE_UNSPECIFIED
	}
}
