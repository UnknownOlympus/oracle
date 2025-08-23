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
	Env           string         `json:"env"`            // Env is the current environment: local, dev, prod.
	Database      PostgresConfig `json:"postgres"`       // Database holds the postgres database configuration
	Token         string         `json:"token"`          // Token is an unique telgram bot token
	PollerTimeout time.Duration  `json:"poller_timeout"` // PollerTimeout its a time which need to close telegram bot poller
	RedisAddr     string         `json:"redis_addr"`     // RedisAddr is the redis server address.
	HermesAddr    string         `json:"hermes_address"` // HermesAddr is the address to grpc server
}

// PostgresConfig struct holds the configuration details for connecting to a PostgreSQL database.
type PostgresConfig struct {
	Host     string `json:"host"`     // Host is the database server address.
	Port     string `json:"port"`     // Port is the database server port.
	User     string `json:"user"`     // User is the database user.
	Password string `json:"password"` // Password is the database user's password.
	Name     string `json:"db_name"`  // Name is the name of the database.
}

// MustLoad loads the configuration from a .env file and returns a Config struct.
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
		RedisAddr:  os.Getenv("REDIS_ADDRESS"),
		HermesAddr: os.Getenv("HERMES_ADDRESS"),
	}
}

func setDeafultEnv(key, override string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = override
	}

	return value
}
