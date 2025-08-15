package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the configuration settings for the application.
// It includes the environment type, database configuration,
// token for authentication, and the timeout duration for polling.
type Config struct {
	Env           string         `yaml:"env"`            // Env is the current environment: local, dev, prod.
	Database      PostgresConfig `yaml:"postgres"`       // Database holds the postgres database configuration
	Token         string         `yaml:"token"`          // Token is an unique telgram bot token
	PollerTimeout time.Duration  `yaml:"poller_timeout"` // PollerTimeout its a time which need to close telegram bot poller
	RedisAddr     string         `yaml:"redis_addr"`     // RedisAddr is the redis server address.
}

// PostgresConfig struct holds the configuration details for connecting to a PostgreSQL database.
type PostgresConfig struct {
	Host     string `yaml:"host"`     // Host is the database server address.
	Port     string `yaml:"port"`     // Port is the database server port.
	User     string `yaml:"user"`     // User is the database user.
	Password string `yaml:"password"` // Password is the database user's password.
	Name     string `yaml:"db_name"`  // Name is the name of the database.
}

// MustLoad loads the configuration from a YAML file and returns a Config struct.
func MustLoad() *Config {
	_ = godotenv.Load()

	timeout, err := time.ParseDuration(setDeafultEnv("ORACLE_TELEGRAM_TIMEOUT", "10s"))
	if err != nil {
		panic("failed to parse interval from configuration")
	}

	return &Config{
		Env:           setDeafultEnv("ORACLE_ENV", "production"),
		Token:         os.Getenv("ORACLE_TELEGRAM_TOKEN"),
		PollerTimeout: timeout,
		Database: PostgresConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USERNAME"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
		},
		RedisAddr: os.Getenv("REDIS_ADDR"),
	}
}

func setDeafultEnv(key, override string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = override
	}

	return value
}
