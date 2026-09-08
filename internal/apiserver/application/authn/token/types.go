package token

import (
	tokendomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/token"
)

type SessionLoader = tokendomain.SessionLoader
type SessionRevoker = tokendomain.SessionRevoker
type SessionExtender = tokendomain.SessionExtender
type SessionRefreshExpirer = tokendomain.SessionRefreshExpirer
type AdmissionPolicy = tokendomain.AdmissionPolicy
type TokenType = tokendomain.TokenType

// TokenClaims 是 application/transport 的兼容投影名；领域声明模型为 AccessTokenClaims。
type TokenClaims = tokendomain.AccessTokenClaims
type AccessTokenClaims = tokendomain.AccessTokenClaims
type ConsumedRefreshToken = tokendomain.ConsumedRefreshToken

const (
	TokenTypeAccess  = tokendomain.TokenTypeAccess
	TokenTypeRefresh = tokendomain.TokenTypeRefresh
)
