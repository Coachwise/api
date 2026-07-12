package payments

// SEP (Saman / سپ) online payment gateway — a RedirectProvider for Toman (IRR).
// Docs: docs/مستند فنی نسخه 3.6.pdf. Flow (base https://sep.shaparak.ir):
//   token:  POST /onlinepg/onlinepg            {action:"token",...}      → {status,token}
//   pay:    GET  /OnlinePG/SendToken?token=...  (browser redirect to bank)
//   callback: gateway POSTs {State,Status,RefNum,ResNum,...} to our RedirectUrl
//   verify: POST /verifyTxnRandomSessionkey/ipg/VerifyTransaction {RefNum,TerminalNumber}
//   reverse:POST /verifyTxnRandomSessionkey/ipg/ReverseTransaction (refund within 50m)
//
// Amounts on the wire are in RIAL, but our system stores IRR as Toman, so we ×10
// outbound (Toman→Rial) and ÷10 the amount returned by Verify. To transact live,
// terminal_id must be set and the server IP whitelisted at Saman.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"coachwise/src/logger"

	"github.com/google/uuid"
)

// sepHTTPClient talks to the gateway. Kept modest so a slow gateway can't hang a
// request past our route timeout.
var sepHTTPClient = &http.Client{Timeout: 20 * time.Second}

type sepProvider struct {
	name       string
	title      string
	currencies []string
	terminalID string
	baseURL    string
}

func (s sepProvider) Name() string         { return s.name }
func (s sepProvider) Title() string        { return s.title }
func (s sepProvider) Currencies() []string { return s.currencies }

func (s sepProvider) Supports(currency string) bool {
	for _, c := range s.currencies {
		if strings.EqualFold(c, currency) {
			return true
		}
	}
	return false
}

// SEP has no synchronous charge — it always needs a browser redirect.
func (s sepProvider) Charge(uuid.UUID, int64, string) (Result, error) {
	return Result{Status: "FAILED"}, ErrRedirectRequired
}

// tomanToRial converts our stored whole-Toman amount to the Rial the gateway wants.
func tomanToRial(toman int64) int64 { return toman * 10 }

func (s sepProvider) GetToken(amount int64, currency, resNum, callbackURL, cellNumber string) (string, string, error) {
	if strings.TrimSpace(s.terminalID) == "" {
		return "", "", fmt.Errorf("%w: SEP terminal_id not configured", ErrGateway)
	}
	reqBody := map[string]any{
		"action":      "token",
		"TerminalId":  s.terminalID,
		"Amount":      tomanToRial(amount),
		"ResNum":      resNum,
		"RedirectUrl": callbackURL,
	}
	if cellNumber != "" {
		reqBody["CellNumber"] = cellNumber
	}
	var out struct {
		Status    int    `json:"status"`
		Token     string `json:"token"`
		ErrorCode string `json:"errorCode"`
		ErrorDesc string `json:"errorDesc"`
	}
	if err := s.postJSON(s.baseURL+"/onlinepg/onlinepg", reqBody, &out); err != nil {
		return "", "", err
	}
	if out.Status != 1 || out.Token == "" {
		logger.Errorf("sep(%s): token request rejected code=%s desc=%s", s.name, out.ErrorCode, out.ErrorDesc)
		return "", "", fmt.Errorf("%w: %s", ErrGateway, firstNonEmpty(out.ErrorDesc, "token request rejected"))
	}
	return s.baseURL + "/OnlinePG/SendToken?token=" + out.Token, out.Token, nil
}

func (s sepProvider) Verify(refNum string) (int64, bool, error) {
	terminal, _ := strconv.ParseInt(s.terminalID, 10, 64)
	reqBody := map[string]any{"RefNum": refNum, "TerminalNumber": terminal}
	var out struct {
		TransactionDetail struct {
			OrginalAmount int64 `json:"OrginalAmount"`
		} `json:"TransactionDetail"`
		ResultCode int  `json:"ResultCode"`
		Success    bool `json:"Success"`
	}
	if err := s.postJSON(s.baseURL+"/verifyTxnRandomSessionkey/ipg/VerifyTransaction", reqBody, &out); err != nil {
		return 0, false, err
	}
	// ResultCode 0 = success, 2 = already verified (idempotent success).
	ok := out.Success && (out.ResultCode == 0 || out.ResultCode == 2)
	// Return the amount in whole Toman to match our stored unit (gateway is Rial).
	return out.TransactionDetail.OrginalAmount / 10, ok, nil
}

func (s sepProvider) postJSON(url string, body any, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := sepHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGateway, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: gateway status %d", ErrGateway, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: bad gateway response: %v", ErrGateway, err)
	}
	return nil
}
