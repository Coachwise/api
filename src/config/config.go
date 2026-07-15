package config

import (
	"log"

	"github.com/spf13/viper"
)

var Config ConfigType

type ConfigType struct {
	Port        int    `mapstructure:"port"`
	Debug       bool   `mapstructure:"debug"`
	Secret      string `mapstructure:"string"`
	OpenAPIPath string `mapstructure:"openapi_path"`
	MediaBaseURL string `mapstructure:"media_base_url"`
	// PublicURL is the externally reachable base URL of this API, used to build
	// the approve/reject capability links sent to Discord (e.g. http://localhost:8000).
	PublicURL string `mapstructure:"public_url"`
	Database    struct {
		URL        string `mapstructure:"url"`
		SqlDir     string `mapstructure:"sqldir"`
		Migrations string `mapstructure:"migrations"`
	} `mapstructure:"database"`
	Storage struct {
		Provider    string   `mapstructure:"provider"`
		Dir         string   `mapstructure:"dir"`
		BaseURL     string   `mapstructure:"base_url"`
		MaxSizeMB   int      `mapstructure:"max_size_mb"`
		AllowedMIME []string `mapstructure:"allowed_mime"`
	} `mapstructure:"storage"`
	CORS struct {
		AllowedOrigins []string `mapstructure:"allowed_origins"`
	} `mapstructure:"cors"`
	Discord struct {
		// ApplicationWebhook receives coach-application notifications with the
		// approve/reject links. When empty, the links are logged instead.
		ApplicationWebhook string `mapstructure:"application_webhook"`
		// AlertWebhook receives panics, 5xx responses, failed jobs and app
		// crashes — the #alert-log channel. When empty, alerts are logged instead.
		AlertWebhook string `mapstructure:"alert_webhook"`
	} `mapstructure:"discord"`
	// Nats is the message queue used to fan events out to consumers (DB-insert
	// now; push / email / SMS later). When URL is empty the queue is disabled and
	// events are dropped — the API still works.
	Nats struct {
		URL string `mapstructure:"url"`
	} `mapstructure:"nats"`
	// SMS gateways for OTP codes, chosen by the recipient's country (dial code) —
	// like the payments providers. Kavenegar handles Iran (+98); SendGrid (or
	// another) covers the rest later. No matching provider → the code is logged
	// (dev). Each entry: type ("kavenegar" | "sendgrid"), the countries (dial
	// codes) it serves, api_key, and (Kavenegar OTP) the verify template name.
	SMS struct {
		Providers []struct {
			Name      string   `mapstructure:"name"`
			Type      string   `mapstructure:"type"`
			Countries []string `mapstructure:"countries"`
			APIKey    string   `mapstructure:"api_key"`
			Sender    string   `mapstructure:"sender"`
			Template  string   `mapstructure:"template"`
			BaseURL   string   `mapstructure:"base_url"`
		} `mapstructure:"providers"`
	} `mapstructure:"sms"`
	// Payment providers for wallet top-ups. Each provider declares the currencies
	// it handles; the buyer picks a currency then one of its providers. Empty list
	// falls back to a stub covering default_currency. Plug Iranian gateways or
	// Stripe by adding entries later.
	Payments struct {
		DefaultCurrency string `mapstructure:"default_currency"`
		// CallbackURL is the PUBLIC URL a redirect gateway (SEP) posts the result
		// to (must be reachable from the internet + IP-whitelisted at the gateway).
		// ReturnURL is the frontend page the user's browser is bounced back to.
		CallbackURL string `mapstructure:"callback_url"`
		ReturnURL   string `mapstructure:"return_url"`
		Providers   []struct {
			Name       string   `mapstructure:"name"`
			Title      string   `mapstructure:"title"`
			Currencies []string `mapstructure:"currencies"`
			APIKey     string   `mapstructure:"api_key"`
			// Type selects the implementation: "" / "stub" (auto-success) or "sep"
			// (Saman redirect gateway). TerminalID + BaseURL configure "sep".
			Type       string `mapstructure:"type"`
			TerminalID string `mapstructure:"terminal_id"`
			BaseURL    string `mapstructure:"base_url"`
			// Logo is a client-served asset path shown in the payment-method picker.
			Logo string `mapstructure:"logo"`
		} `mapstructure:"providers"`
	} `mapstructure:"payments"`
}

func Init(configPath string) {
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("Config file not found: %s", err)
		} else {
			log.Fatalf("Error reading config file: %s", err)
		}
	}

	if err := viper.Unmarshal(&Config); err != nil {
		log.Fatal(err)
	}

	log.Printf("Using config file: %s\n", viper.ConfigFileUsed())
}
