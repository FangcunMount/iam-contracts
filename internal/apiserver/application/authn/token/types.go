package token

import (
	tokendomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/token"
)

type SessionLoader = tokendomain.SessionLoader
type SessionRevoker = tokendomain.SessionRevoker
type SessionExtender = tokendomain.SessionExtender
type SessionRefreshExpirer = tokendomain.SessionRefreshExpirer
type AdmissionPolicy = tokendomain.AdmissionPolicy
type TokenType = tokendomain.TokenType

// TokenClaims 是 application/transport 的兼容投影名；领域事实名为 VerifiedTokenClaims。
type TokenClaims = tokendomain.VerifiedTokenClaims
type VerifiedTokenClaims = tokendomain.VerifiedTokenClaims
type ConsumedRefreshToken = tokendomain.ConsumedRefreshToken

const (
	TokenTypeAccess  = tokendomain.TokenTypeAccess
	TokenTypeRefresh = tokendomain.TokenTypeRefresh
)
