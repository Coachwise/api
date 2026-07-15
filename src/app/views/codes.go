package views

import "coachwise/src/errcode"

// The error-code system lives in the shared errcode package (so the auth
// middleware can use it too). These aliases let view handlers keep calling
// Abort / AbortStatus / CodeX unqualified.

type ErrCode = errcode.ErrCode

const (
	CodeUnknown      = errcode.CodeUnknown
	CodeValidation   = errcode.CodeValidation
	CodeBadRequest   = errcode.CodeBadRequest
	CodeUnauthorized = errcode.CodeUnauthorized
	CodeForbidden    = errcode.CodeForbidden
	CodeNotFound     = errcode.CodeNotFound
	CodeRateLimited  = errcode.CodeRateLimited
	CodeConflict     = errcode.CodeConflict

	CodeInvalidCredentials = errcode.CodeInvalidCredentials
	CodeOTPInvalid         = errcode.CodeOTPInvalid
	CodeOTPCooldown        = errcode.CodeOTPCooldown
	CodeRefreshInvalid     = errcode.CodeRefreshInvalid
	CodeEmailExists        = errcode.CodeEmailExists
	CodeUsernameExists     = errcode.CodeUsernameExists
	CodeUserNotFound       = errcode.CodeUserNotFound
	CodeOTPSendFailed      = errcode.CodeOTPSendFailed
	CodeTokenGeneration    = errcode.CodeTokenGeneration

	CodeNotOwner           = errcode.CodeNotOwner
	CodeCoachOnly          = errcode.CodeCoachOnly
	CodeSelfAction         = errcode.CodeSelfAction
	CodeNotConnected       = errcode.CodeNotConnected
	CodePlanLimit          = errcode.CodePlanLimit
	CodePackageUnavailable = errcode.CodePackageUnavailable

	CodeInsufficientFunds      = errcode.CodeInsufficientFunds
	CodeCurrencyMismatch       = errcode.CodeCurrencyMismatch
	CodePriceNotConfigured     = errcode.CodePriceNotConfigured
	CodeInvalidDuration        = errcode.CodeInvalidDuration
	CodePayoutExceedsAvailable = errcode.CodePayoutExceedsAvailable
	CodePaymentFailed          = errcode.CodePaymentFailed
	CodeUnsupportedCurrency    = errcode.CodeUnsupportedCurrency
	CodeNoProvider             = errcode.CodeNoProvider
	CodePayoutAccountMissing   = errcode.CodePayoutAccountMissing
	CodeBalanceNegative        = errcode.CodeBalanceNegative
)

var (
	Abort       = errcode.Abort
	AbortMsg    = errcode.AbortMsg
	AbortStatus = errcode.AbortStatus
	AbortServer = errcode.AbortServer
)
