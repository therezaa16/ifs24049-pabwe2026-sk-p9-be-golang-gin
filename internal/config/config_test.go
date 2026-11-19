package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigDefaults(t *testing.T) {
	for _, k := range []string{
		"APP_ENV", "APP_PORT", "APP_KEY", "ADMIN_USERNAME", "ADMIN_PASSWORD",
		"DB_CONNECTION", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD", "DB_SSLMODE",
		"SEARCH_ENGINE_ENDPOINT", "OPENAI_ENDPOINT", "OPENAI_KEY", "OPENAI_MODEL",
	} {
		t.Setenv(k, "")
	}
	cfg := LoadConfig()
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "pabwe_p9_blog", cfg.DBDatabase)
	assert.Equal(t, "https://delcom.org/api/v1/search", cfg.SearchEngineEndpoint)
	assert.Equal(t, "admin", cfg.AdminUsername)
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("APP_PORT", " 9000 ")
	t.Setenv("DB_HOST", "db.local")
	cfg := LoadConfig()
	assert.Equal(t, "9000", cfg.AppPort)
	assert.Equal(t, "db.local", cfg.DBHost)
}

func TestDSN(t *testing.T) {
	cfg := &Config{DBHost: "h", DBUsername: "u", DBPassword: "p", DBDatabase: "d", DBPort: "1", DBSSLMode: "disable"}
	assert.Equal(t, "host=h user=u password=p dbname=d port=1 sslmode=disable TimeZone=Asia/Jakarta", cfg.DSN())
}
