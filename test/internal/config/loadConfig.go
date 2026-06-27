package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Cfg holds all runtime configuration for the migrator test app.
// Only app-level and Postgres fields are included; auth/cache/object-store
// fields have been removed as this service is purely a migration testing ground.
type Cfg struct {
	AppEnv     string
	AppName    string
	AppVersion string

	AppPort  string
	AppDebug bool

	LogLevel string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBURL      string
}

// LoadConfig reads the .env file at path (if present) and returns a populated Cfg.
// Missing keys fall back to the defaults defined below.
func LoadConfig(path string) (Cfg, error) {
	err := godotenv.Load(path)
	if err != nil && !os.IsNotExist(err) {
		return Cfg{}, err
	}

	cfg := Cfg{
		AppEnv:     getEnv("APP_ENV", "development"),
		AppName:    getEnv("APP_NAME", "migrator-test"),
		AppVersion: getEnv("APP_VERSION", "0.0.1"),

		AppPort:  getEnv("APP_PORT", "8484"),
		AppDebug: getEnvAsBool("APP_DEBUG", false),

		LogLevel: getEnv("APP_LOG_LEVEL", "info"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", ""),
		DBURL:      getEnv("DATABASE_URL", ""),
	}

	return cfg, nil
}

func getEnv(key string, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultValue
	}
	return b
}
