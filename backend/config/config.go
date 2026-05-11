package config

import "os"

type Config struct {
	AsimutEmail    string
	AsimutPassword string
	AppPassword    string
	DatabasePath   string
	Port           string
}

func Load() *Config {
	return &Config{
		AsimutEmail:    os.Getenv("ASIMUT_EMAIL"),
		AsimutPassword: os.Getenv("ASIMUT_PASSWORD"),
		AppPassword:    os.Getenv("APP_PASSWORD"),
		DatabasePath:   getEnvOrDefault("DATABASE_PATH", "/data/asimut.db"),
		Port:           getEnvOrDefault("PORT", "8080"),
	}
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
