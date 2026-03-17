package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv     string
	AppHost    string
	AppPort    int
	AppDebug   bool
	SecretKey  string
	DBPath     string
	UploadsDir string
	PublicDir  string
	Theme      string
}

func Load(envPath string) (*Config, error) {
	if envPath != "" {
		_ = godotenv.Load(envPath)
	} else {
		_ = godotenv.Load()
	}

	cfg := &Config{
		AppEnv:     getEnv("APP_ENV", "production"),
		AppHost:    getEnv("APP_HOST", "localhost"),
		AppPort:    getEnvInt("APP_PORT", 3000),
		AppDebug:   getEnvBool("APP_DEBUG", false),
		SecretKey:  getEnv("SECRET_KEY", ""),
		DBPath:     getEnv("DB_PATH", "./storage/press.db"),
		UploadsDir: getEnv("UPLOADS_DIR", "./storage/uploads"),
		PublicDir:  getEnv("PUBLIC_DIR", "./public"),
		Theme:      getEnv("THEME", "default"),
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func (c *Config) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.AppHost, c.AppPort)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return parsed
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return parsed
	}
	return defaultValue
}
