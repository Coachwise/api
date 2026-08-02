// Package errcode is the single source of truth for API error codes. It lives as
// a leaf package so both the auth middleware and the view handlers can emit coded
// responses without an import cycle. The frontend maps each number to a localized
// message (see app/src/api/errors.ts). Never renumber an existing code.
package errcode

import (
	"database/sql"
	"errors"
	"net/http"

	"coachwise/src/logger"

	"github.com/gin-gonic/gin"
)

// ErrCode is a stable, numeric error identifier returned to clients as `code`.
// Ranges: 1000–1099 generic/transport, 1100–1199 auth, 1200–1299 domain rules.
type ErrCode int

const (
	// Generic
	CodeUnknown      ErrCode = 1000
	CodeValidation   ErrCode = 1001
	CodeBadRequest   ErrCode = 1002
	CodeUnauthorized ErrCode = 1003
	CodeForbidden    ErrCode = 1004
	CodeNotFound     ErrCode = 1005
	CodeRateLimited  ErrCode = 1006
	CodeConflict     ErrCode = 1007

	// Auth
	CodeInvalidCredentials ErrCode = 1100
	CodeOTPInvalid         ErrCode = 1101
	CodeOTPCooldown        ErrCode = 1102
	CodeRefreshInvalid     ErrCode = 1103
	CodeEmailExists        ErrCode = 1104
	CodeUsernameExists     ErrCode = 1105
	CodeUserNotFound       ErrCode = 1106
	CodeOTPSendFailed      ErrCode = 1107
	CodeTokenGeneration    ErrCode = 1108

	// Domain business rules
	CodeNotOwner           ErrCode = 1200
	CodeCoachOnly          ErrCode = 1201
	CodeSelfAction         ErrCode = 1202
	CodeNotConnected       ErrCode = 1203
	CodePlanLimit          ErrCode = 1204
	CodePackageUnavailable ErrCode = 1205
	CodeTicketNotYourTurn  ErrCode = 1206
	CodeTicketClosed       ErrCode = 1207

	// Billing / wallet (1300+)
	CodeInsufficientFunds      ErrCode = 1300
	CodeCurrencyMismatch       ErrCode = 1301
	CodePriceNotConfigured     ErrCode = 1302
	CodeInvalidDuration        ErrCode = 1303
	CodePayoutExceedsAvailable ErrCode = 1304
	CodePaymentFailed          ErrCode = 1305
	CodeUnsupportedCurrency    ErrCode = 1306
	CodeNoProvider             ErrCode = 1307
	CodePayoutAccountMissing   ErrCode = 1308
	// The coach owes money back: a refund clawed back more than was left in the
	// wallet. Withdrawals wait until that's settled.
	CodeBalanceNegative ErrCode = 1309

	// AI assistant (1400+)
	CodeAIDisabled ErrCode = 1400 // no model configured
	CodeAIFailed   ErrCode = 1401 // the turn errored or produced no answer
)

type meta struct {
	status int
	msg    string // developer-facing fallback (clients localize by code)
}

var codeMeta = map[ErrCode]meta{
	CodeUnknown:      {http.StatusInternalServerError, "Something went wrong"},
	CodeValidation:   {http.StatusBadRequest, "Invalid request"},
	CodeBadRequest:   {http.StatusBadRequest, "Invalid request"},
	CodeUnauthorized: {http.StatusUnauthorized, "Authentication required"},
	CodeForbidden:    {http.StatusForbidden, "Not allowed"},
	CodeNotFound:     {http.StatusNotFound, "Not found"},
	CodeRateLimited:  {http.StatusTooManyRequests, "Too many requests, please wait"},
	CodeConflict:     {http.StatusConflict, "Conflict"},

	CodeInvalidCredentials: {http.StatusBadRequest, "Email or password is incorrect"},
	CodeOTPInvalid:         {http.StatusBadRequest, "The code is invalid or expired"},
	CodeOTPCooldown:        {http.StatusTooManyRequests, "Please wait before requesting another code"},
	CodeRefreshInvalid:     {http.StatusUnauthorized, "Invalid refresh token"},
	CodeEmailExists:        {http.StatusBadRequest, "Email already exists"},
	CodeUsernameExists:     {http.StatusBadRequest, "Username already exists"},
	CodeUserNotFound:       {http.StatusNotFound, "User not found"},
	CodeOTPSendFailed:      {http.StatusBadRequest, "Couldn't send the code"},
	CodeTokenGeneration:    {http.StatusInternalServerError, "Couldn't issue a session"},

	CodeNotOwner:           {http.StatusForbidden, "Only the owner can do this"},
	CodeCoachOnly:          {http.StatusForbidden, "Only coaches can do this"},
	CodeSelfAction:         {http.StatusBadRequest, "You can't do that to yourself"},
	CodeNotConnected:       {http.StatusForbidden, "You need to be connected first"},
	CodePlanLimit:          {http.StatusForbidden, "You've reached your plan limit"},
	CodePackageUnavailable: {http.StatusBadRequest, "This package isn't available"},
	CodeTicketNotYourTurn:  {http.StatusConflict, "Please wait for support to reply before sending again"},
	CodeTicketClosed:       {http.StatusConflict, "This ticket is closed"},

	CodeInsufficientFunds:      {http.StatusBadRequest, "Insufficient wallet balance"},
	CodeCurrencyMismatch:       {http.StatusBadRequest, "Currency mismatch"},
	CodePriceNotConfigured:     {http.StatusBadRequest, "Pricing is not configured"},
	CodeInvalidDuration:        {http.StatusBadRequest, "Unsupported duration"},
	CodePayoutExceedsAvailable: {http.StatusBadRequest, "Payout exceeds available balance"},
	CodePaymentFailed:          {http.StatusBadRequest, "Payment failed"},
	CodeUnsupportedCurrency:    {http.StatusBadRequest, "Unsupported currency"},
	CodeNoProvider:             {http.StatusBadRequest, "Payment method not available"},
	CodePayoutAccountMissing:   {http.StatusBadRequest, "Set up a payout account first"},
	CodeBalanceNegative:        {http.StatusBadRequest, "Your balance is negative after a refund; it must be settled before a payout"},

	CodeAIDisabled: {http.StatusServiceUnavailable, "AI assistant is not available"},
	CodeAIFailed:   {http.StatusInternalServerError, "The assistant couldn't complete this"},
}

var statusCode = map[int]ErrCode{
	http.StatusBadRequest:          CodeBadRequest,
	http.StatusUnauthorized:        CodeUnauthorized,
	http.StatusForbidden:           CodeForbidden,
	http.StatusNotFound:            CodeNotFound,
	http.StatusConflict:            CodeConflict,
	http.StatusTooManyRequests:     CodeRateLimited,
	http.StatusInternalServerError: CodeUnknown,
}

// Abort writes a coded error response using the code's default message.
func Abort(c *gin.Context, code ErrCode) {
	m, ok := codeMeta[code]
	if !ok {
		m = codeMeta[CodeUnknown]
	}
	c.AbortWithStatusJSON(m.status, gin.H{"code": int(code), "error": m.msg})
}

// AbortMsg writes a coded error response with a custom developer-facing message.
func AbortMsg(c *gin.Context, code ErrCode, msg string) {
	m, ok := codeMeta[code]
	if !ok {
		m = codeMeta[CodeUnknown]
	}
	c.AbortWithStatusJSON(m.status, gin.H{"code": int(code), "error": msg})
}

// AbortStatus attaches a generic code derived from the HTTP status, keeping the
// original developer message. For transport-level errors (bad id, not found).
func AbortStatus(c *gin.Context, status int, msg string) {
	code, ok := statusCode[status]
	if !ok {
		code = CodeUnknown
	}
	c.AbortWithStatusJSON(status, gin.H{"code": int(code), "error": msg})
}

// AbortServer logs an internal/db error (traceable by method + path) and returns
// a generic coded response so raw driver/query strings never reach the client.
func AbortServer(c *gin.Context, err error) {
	// A lookup that found nothing is not a server fault — it's a 404. Handlers
	// pass fetch errors straight here, so catching it once covers all of them.
	if errors.Is(err, sql.ErrNoRows) {
		Abort(c, CodeNotFound)
		return
	}
	logger.WithFields(logger.Fields{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"code":   int(CodeUnknown),
	}).Error(err)
	m := codeMeta[CodeUnknown]
	c.AbortWithStatusJSON(m.status, gin.H{"code": int(CodeUnknown), "error": m.msg})
}
