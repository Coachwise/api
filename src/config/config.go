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
	CORS struct {
		AllowedOrigins []string `mapstructure:"allowed_origins"`
	} `mapstructure:"cors"`
	Discord struct {
		// ApplicationWebhook receives coach-application notifications with the
		// approve/reject links. When empty, the links are logged instead.
		ApplicationWebhook string `mapstructure:"application_webhook"`
	} `mapstructure:"discord"`
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
