package config_test

import (
	"testing"
	"time"

	"github.com/UnknownOlympus/oracle/internal/config"
	"github.com/stretchr/testify/assert"
)

func Test_MustLoadFromFile(t *testing.T) {
	t.Setenv("ORACLE_ENV", "local")
	t.Setenv("ORACLE_TELEGRAM_TOKEN", "someTelegramToken")
	t.Setenv("DB_HOST", "testHost")
	t.Setenv("DB_PORT", "12345")
	t.Setenv("DB_USERNAME", "admin")
	t.Setenv("DB_PASSWORD", "adminpass")
	t.Setenv("DB_NAME", "testName")

	cfg := config.MustLoad()

	assert.Equal(t, "local", cfg.Env)
	assert.Equal(t, "someTelegramToken", cfg.Token)
	assert.Equal(t, 10*time.Second, cfg.PollerTimeout)
	assert.Equal(t, "testHost", cfg.Database.Host)
	assert.Equal(t, "12345", cfg.Database.Port)
	assert.Equal(t, "admin", cfg.Database.User)
	assert.Equal(t, "adminpass", cfg.Database.Password)
	assert.Equal(t, "testName", cfg.Database.Name)
}

func TestMustLoad_IntervalError(t *testing.T) {
	t.Setenv("ORACLE_TELEGRAM_TIMEOUT", "error_value")

	assert.PanicsWithValue(t, "failed to parse interval from configuration", func() {
		config.MustLoad()
	})
}
