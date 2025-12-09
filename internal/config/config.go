package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL      string `envconfig:"DATABASE_URL" required:"true"`
	CurseForgeAPIKey string `envconfig:"CURSEFORGE_API_KEY" required:"true"`
	Environment      string `envconfig:"ENV" default:"development"`
}

func Load() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
