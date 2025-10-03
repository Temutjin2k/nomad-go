package config

import (
	"errors"
	"flag"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/configparser"
)

// Flags
var (
	modeFlag = flag.String("mode", "", "application mode")
)

// Errors
var (
	ErrModeNotProvided = errors.New("mode flag not provided")
)

// Config contains all configuration variables of the application
type Config struct {
	Mode types.ServiceMode

	Database  DatabaseConfig
	RabbitMQ  RabbitMQConfig
	WebSocket WebSocketConfig
	Services  ServicesConfig
}

type DatabaseConfig struct {
	Host     string `env:"DATABASE_HOST" default:"localhost"`
	Port     string `env:"DATABASE_PORT" default:"5432"`
	User     string `env:"DATABASE_USER" default:"ridehail_user"`
	Password string `env:"DATABASE_PASSWORD" default:"ridehail_pass"`
	Database string `env:"DATABASE_DATABASE" default:"ridehail_db"`
}

func (c DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

type RabbitMQConfig struct {
	Host     string `env:"RABBITMQ_HOST" default:"localhost"`
	Port     string `env:"RABBITMQ_PORT" default:"5672"`
	User     string `env:"RABBITMQ_USER" default:"guest"`
	Password string `env:"RABBITMQ_PASSWORD" default:"guest"`
}

func (c RabbitMQConfig) GetDSN() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/",
		c.User,
		c.Password,
		c.Host,
		c.Port,
	)
}

type WebSocketConfig struct {
	Port string `env:"WEBSOCKET_PORT" default:"8080"`
}

type ServicesConfig struct {
	RideService           string `env:"SERVICES_RIDE_SERVICE" default:"3000"`
	DriverLocationService string `env:"SERVICES_DRIVER_LOCATION_SERVICE" default:"3001"`
	AdminService          string `env:"SERVICES_ADMIN_SERVICE" default:"3004"`
}

func NewConfig(filepath string) (*Config, error) {
	cfg := &Config{}

	// Loading enviromental variables and parsing to config struct.
	if err := configparser.LoadAndParseYaml(filepath, cfg); err != nil {
		return nil, fmt.Errorf("failed to load and parse config: %w", err)
	}

	// Parsing flags
	if err := parseFlags(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	return cfg, nil
}

func parseFlags(cfg *Config) error {
	if modeFlag == nil || *modeFlag == "" {
		return ErrModeNotProvided
	}

	cfg.Mode = types.ServiceMode(*modeFlag)

	return nil
}
