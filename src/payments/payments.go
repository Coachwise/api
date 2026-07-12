// Package payments abstracts money-in (top-up) behind a provider registry. Each
// provider declares which currencies it handles, so a purchase can offer the
// buyer the providers valid for their chosen currency. In beta every provider is
// a stub that auto-succeeds; real Iranian gateways and Stripe implement Provider
// later and are added via config (mirrors src/sms).
package payments

import (
	"errors"
	"strings"

	"coachwise/src/config"
	"coachwise/src/logger"

	"github.com/google/uuid"
)

var (
	// ErrNoProvider means the requested provider isn't registered.
	ErrNoProvider = errors.New("payment provider not available")
	// ErrCurrencyUnsupported means the provider doesn't handle that currency.
	ErrCurrencyUnsupported = errors.New("provider does not support this currency")
	// ErrRedirectRequired means the provider can't charge synchronously — the
	// buyer must be redirected to the gateway (use the RedirectProvider methods).
	ErrRedirectRequired = errors.New("provider requires redirect")
	// ErrGateway wraps a failure talking to an external gateway.
	ErrGateway = errors.New("payment gateway error")
)

// Result is the outcome of a charge attempt.
type Result struct {
	Ref    string // provider reference to persist on the payment row
	Status string // PAID | PENDING | FAILED
}

// Descriptor is the public, client-facing view of a provider (for pickers).
type Descriptor struct {
	Name       string   `json:"name"`
	Title      string   `json:"title"`
	Currencies []string `json:"currencies"`
	// Redirect is true when paying with this provider sends the buyer to a hosted
	// gateway page (so the client uses the top-up/initiate + redirect flow).
	Redirect bool `json:"redirect"`
	// Logo is a client-served asset path (e.g. "/sep.png"), if configured.
	Logo string `json:"logo,omitempty"`
}

type Provider interface {
	Name() string
	Title() string
	Currencies() []string
	Supports(currency string) bool
	Charge(userID uuid.UUID, amount int64, currency string) (Result, error)
}

// stub auto-succeeds for its configured currencies (beta placeholder).
type stub struct {
	name       string
	title      string
	currencies []string
}

func (s stub) Name() string         { return s.name }
func (s stub) Title() string        { return s.title }
func (s stub) Currencies() []string { return s.currencies }

func (s stub) Supports(currency string) bool {
	for _, c := range s.currencies {
		if strings.EqualFold(c, currency) {
			return true
		}
	}
	return false
}

func (s stub) Charge(userID uuid.UUID, amount int64, currency string) (Result, error) {
	if !s.Supports(currency) {
		return Result{Status: "FAILED"}, ErrCurrencyUnsupported
	}
	ref := "stub_" + uuid.NewString()
	logger.Infof("payments(%s): charged %d %s for user=%s ref=%s", s.name, amount, currency, userID, ref)
	return Result{Ref: ref, Status: "PAID"}, nil
}

// registry holds the active providers, keyed by name (insertion order preserved
// separately so pickers are deterministic).
var (
	registry = map[string]Provider{}
	order    []string
	// logos maps provider name → client-served logo path (from config).
	logos = map[string]string{}
)

// Init builds the provider registry from config (call once at startup). When no
// providers are configured, a single stub covering the default currency is used
// so beta works out of the box.
func Init() {
	registry = map[string]Provider{}
	order = nil
	logos = map[string]string{}

	for _, p := range config.Config.Payments.Providers {
		if p.Logo != "" {
			logos[p.Name] = p.Logo
		}
		var prov Provider
		switch strings.ToLower(p.Type) {
		case "sep":
			prov = sepProvider{
				name:       p.Name,
				title:      firstNonEmpty(p.Title, p.Name),
				currencies: p.Currencies,
				terminalID: p.TerminalID,
				baseURL:    strings.TrimRight(firstNonEmpty(p.BaseURL, "https://sep.shaparak.ir"), "/"),
			}
		default:
			prov = stub{name: p.Name, title: firstNonEmpty(p.Title, p.Name), currencies: p.Currencies}
		}
		register(prov)
	}
	if len(order) == 0 {
		def := config.Config.Payments.DefaultCurrency
		if def == "" {
			def = "IRR"
		}
		register(stub{name: "stub", title: "Test Gateway", currencies: []string{def}})
	}
}

func register(p Provider) {
	if _, exists := registry[p.Name()]; !exists {
		order = append(order, p.Name())
	}
	registry[p.Name()] = p
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// ProvidersFor returns the descriptors of providers that handle a currency, in
// registration order.
func ProvidersFor(currency string) []Descriptor {
	out := []Descriptor{}
	for _, name := range order {
		p := registry[name]
		if p.Supports(currency) {
			out = append(out, Descriptor{Name: p.Name(), Title: p.Title(), Currencies: p.Currencies(), Redirect: IsRedirect(p.Name()), Logo: logos[p.Name()]})
		}
	}
	return out
}

// HasProviderFor reports whether any provider handles the currency.
func HasProviderFor(currency string) bool {
	return len(ProvidersFor(currency)) > 0
}

// Charge routes a money-in request to the named provider, validating currency.
func Charge(providerName string, userID uuid.UUID, amount int64, currency string) (Result, error) {
	p, ok := registry[providerName]
	if !ok {
		return Result{Status: "FAILED"}, ErrNoProvider
	}
	if !p.Supports(currency) {
		return Result{Status: "FAILED"}, ErrCurrencyUnsupported
	}
	return p.Charge(userID, amount, currency)
}

// --- Redirect (hosted-page) providers ----------------------------------------

// RedirectProvider is a gateway the buyer must be sent to (a bank page), rather
// than charged server-to-server. The flow is: GetToken → redirect the browser →
// the gateway calls our callback → Verify. Amounts in/out are in the currency's
// whole unit (Toman for IRR); the provider converts to the gateway's unit itself.
// Concrete gateways live in their own files (e.g. sep.go).
type RedirectProvider interface {
	Provider
	// GetToken opens a gateway session and returns the URL to send the browser to.
	// resNum is our unique reference; callbackURL is where the gateway posts back.
	GetToken(amount int64, currency, resNum, callbackURL, cellNumber string) (redirectURL, ref string, err error)
	// Verify confirms a returned transaction by its RefNum, returning the settled
	// amount in the currency's whole unit and whether it succeeded.
	Verify(refNum string) (amount int64, ok bool, err error)
}

func redirectProvider(name string) (RedirectProvider, bool) {
	p, ok := registry[name]
	if !ok {
		return nil, false
	}
	rp, ok := p.(RedirectProvider)
	return rp, ok
}

// IsRedirect reports whether the named provider needs a browser redirect.
func IsRedirect(name string) bool {
	_, ok := redirectProvider(name)
	return ok
}

// GetToken starts a redirect payment and returns the URL to send the browser to.
func GetToken(name string, amount int64, currency, resNum, callbackURL, cellNumber string) (redirectURL, ref string, err error) {
	rp, ok := redirectProvider(name)
	if !ok {
		return "", "", ErrNoProvider
	}
	if !rp.Supports(currency) {
		return "", "", ErrCurrencyUnsupported
	}
	return rp.GetToken(amount, currency, resNum, callbackURL, cellNumber)
}

// Verify confirms a returned redirect transaction.
func Verify(name, refNum string) (amount int64, ok bool, err error) {
	rp, has := redirectProvider(name)
	if !has {
		return 0, false, ErrNoProvider
	}
	return rp.Verify(refNum)
}
