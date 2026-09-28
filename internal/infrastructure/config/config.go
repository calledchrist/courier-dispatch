package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	App  AppConfig
	HTTP HTTPConfig
}

type AppConfig struct {
	Environment string
}

type HTTPConfig struct {
	Host string
	Port string
}

// Load resolves process environment variables, then .env, then defaults.
func Load() (*Config, error) {
	settings := viper.New()
	settings.SetConfigFile(".env")
	settings.SetConfigType("env")
	settings.AutomaticEnv()

	// Dotenv files contain flat keys. Use the same keys for every source so
	// file values and process environment variables have identical behavior.
	settings.SetDefault("COURIER_APP_ENVIRONMENT", "development")
	settings.SetDefault("COURIER_HTTP_HOST", "0.0.0.0")
	settings.SetDefault("COURIER_HTTP_PORT", "8080")

	if err := settings.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
			return nil, err
		}
	}

	return &Config{
		App: AppConfig{
			Environment: settings.GetString("COURIER_APP_ENVIRONMENT"),
		},
		HTTP: HTTPConfig{
			Host: settings.GetString("COURIER_HTTP_HOST"),
			Port: settings.GetString("COURIER_HTTP_PORT"),
		},
	}, nil
}
