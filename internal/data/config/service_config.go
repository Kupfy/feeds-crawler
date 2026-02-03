package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type ServiceConfig struct {
	ServerHost            string        `envconfig:"HOST" default:"0.0.0.0"`
	ServerPort            string        `envconfig:"SERVER_PORT" default:"8080"`
	ENV                   string        `envconfig:"ENV" default:"local"`
	JwtSecret             string        `envconfig:"JWT_SECRET"`
	DbHost                string        `envconfig:"DB_HOST" default:"postgres"`
	DBPort                string        `envconfig:"DB_PORT" default:"5432"`
	DbUser                string        `envconfig:"DB_USER" default:"postgres"`
	DbPassword            string        `envconfig:"DB_PASSWORD" default:""`
	DbName                string        `envconfig:"DB_NAME" default:"postgres"`
	RedisHost             string        `envconfig:"REDIS_HOST" default:"redis"`
	RedisPort             string        `envconfig:"REDIS_PORT" default:"6379"`
	IngredientServiceAddr string        `envconfig:"INGREDIENT_SERVICE_ADDR" default:"ingredient-parser:50051"`
	QueueTimeout          time.Duration `envconfig:"QUEUE_TIMEOUT_SECONDS" default:"30s"`
	DefaultConcurrency    int           `envconfig:"DEFAULT_CONCURRENCY" default:"8"`
	DefaultMaxDepth       int           `envconfig:"DEFAULT_MAX_DEPTH" default:"5"`
	DefaultRetries        int           `envconfig:"DEFAULT_RETRIES" default:"3"`
	DefaultBackoff        time.Duration `envconfig:"DEFAULT_BACKOFF" default:"1000000s"`
	FeatureFlags          FeatureFlags  `envconfig:"FEATURE_FLAGS"`
}

func NewServiceConfig(serviceConf *ServiceConfig) ServiceConfig {
	err := envconfig.Process("", serviceConf)
	checkConfig(err)
	return *serviceConf
}

func checkConfig(cfgErr error) {
	if cfgErr != nil {
		slog.Error("Unable to map service configuration", slog.Any("err", cfgErr.Error()))

		os.Exit(-1)
	}
}

// Load reads configuration from environment variables
func Load() (ServiceConfig, error) {
	var cfg ServiceConfig
	err := envconfig.Process("", &cfg)
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

// Address returns the server address string
func (c *ServiceConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.ServerHost, c.ServerPort)
}

// IsDevelopment returns true if running in development mode
func (c *ServiceConfig) IsDevelopment() bool {
	return c.ENV == "development" || c.ENV == "local"
}
