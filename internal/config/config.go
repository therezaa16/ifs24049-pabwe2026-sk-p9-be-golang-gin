// Package config memuat konfigurasi aplikasi dari environment / berkas .env.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config menampung seluruh konfigurasi runtime.
type Config struct {
	AppEnv  string
	AppPort string
	AppKey  string

	AdminUsername string
	AdminPassword string

	DBConnection string
	DBHost       string
	DBPort       string
	DBDatabase   string
	DBUsername   string
	DBPassword   string
	DBSSLMode    string

	SearchEngineEndpoint string
	OpenAIEndpoint       string
	OpenAIKey            string
	OpenAIModel          string
}

func get(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// LoadConfig membaca .env (jika ada) lalu variabel lingkungan dengan nilai default.
func LoadConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		AppEnv:  get("APP_ENV", "development"),
		AppPort: get("APP_PORT", "8081"),
		AppKey:  get("APP_KEY", "delcom_blog_secret_key_super_secure_2026_random_string_32chars"),

		AdminUsername: get("ADMIN_USERNAME", "admin"),
		AdminPassword: get("ADMIN_PASSWORD", "admin"),

		DBConnection: get("DB_CONNECTION", "pgsql"),
		DBHost:       get("DB_HOST", "127.0.0.1"),
		DBPort:       get("DB_PORT", "5432"),
		DBDatabase:   get("DB_DATABASE", "pabwe_p9_blog"),
		DBUsername:   get("DB_USERNAME", "postgres"),
		DBPassword:   get("DB_PASSWORD", "postgres"),
		DBSSLMode:    get("DB_SSLMODE", "disable"),

		SearchEngineEndpoint: get("SEARCH_ENGINE_ENDPOINT", "https://delcom.org/api/v1/search"),
		OpenAIEndpoint:       get("OPENAI_ENDPOINT", "https://delcom.org/api/v1"),
		OpenAIKey:            get("OPENAI_KEY", "key"),
		OpenAIModel:          get("OPENAI_MODEL", "nama-model"),
	}
}

// DSN menyusun connection string PostgreSQL.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		c.DBHost, c.DBUsername, c.DBPassword, c.DBDatabase, c.DBPort, c.DBSSLMode,
	)
}
