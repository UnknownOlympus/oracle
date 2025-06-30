package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds the configuration settings for the application.
// It includes the environment type, database configuration,
// token for authentication, and the timeout duration for polling.
type Config struct {
	Env           string         `env-default:"local" yaml:"env"`                                // Env is the current environment: local, dev, prod.
	Database      PostgresConfig `                    yaml:"postgres"       env-required:"true"` // Database holds the postgres database configuration
	Token         string         `                    yaml:"token"          env-required:"true"` // Token is an unique telgram bot token
	PollerTimeout time.Duration  `env-default:"10s"   yaml:"poller_timeout"`                     // PollerTimeout its a time which need to close telegram bot poller
}

// PostgresConfig struct holds the configuration details for connecting to a PostgreSQL database.
type PostgresConfig struct {
	Host     string `yaml:"host"`                        // Host is the database server address.
	Port     string `yaml:"port"     env-default:"5432"` // Port is the database server port.
	User     string `yaml:"user"`                        // User is the database user.
	Password string `yaml:"password"`                    // Password is the database user's password.
	Name     string `yaml:"db_name"`                     // Name is the name of the database.
}

// MustLoad loads the configuration from a YAML file and returns a Config struct.
func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		panic("config path is empty")
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		panic("config error: " + err.Error())
	}

	defPollerTimeout := 10

	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("telegram.timeout", time.Duration(defPollerTimeout*int(time.Second)))

	return &Config{
		Env:           viper.GetString("env"),
		Token:         viper.GetString("telegram.token"),
		PollerTimeout: viper.GetDuration("telegram.timeout"),
		Database: PostgresConfig{
			Host:     viper.GetString("postgres.host"),
			Port:     viper.GetString("postgres.port"),
			User:     viper.GetString("postgres.user"),
			Password: viper.GetString("postgres.password"),
			Name:     viper.GetString("postgres.db_name"),
		},
	}
}
