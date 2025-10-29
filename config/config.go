package config

import (
	"errors"
	"flag"
	"fmt"
	"time"

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
type (
	Config struct {
		Mode types.ServiceMode

		Database          DatabaseConfig
		RabbitMQ          RabbitMQConfig
		WebSocket         WebSocketConfig
		ExternalAPIConfig ExternalAPIConfig
		Services          ServicesConfig
		Auth              Auth
	}

	DatabaseConfig struct {
		Host     string `env:"DATABASE_HOST" default:"localhost"`
		Port     string `env:"DATABASE_PORT" default:"5432"`
		User     string `env:"DATABASE_USER" default:"ridehail_user"`
		Password string `env:"DATABASE_PASSWORD" default:"ridehail_pass"`
		Database string `env:"DATABASE_DATABASE" default:"ridehail_db"`

		MaxOpenConns int32  `env:"DATABASE_MAXOPENCONN" default:"25"`
		MaxIdleTime  string `env:"DATABASE_MAXIDLETIME" default:"15m"`

		MaxConns        int32         `env:"DATABASE_MAXCONNS" default:"20"`         // максимум открытых соединений
		MinConns        int32         `env:"DATABASE_MINCONNS" default:"2"`          // минимум соединений в пуле
		MaxConnLifetime time.Duration `env:"DATABASE_MAXCONNLIFETIME" default:"30m"` // макс. "время жизни" соединения
		MaxConnIdleTime time.Duration `env:"DATABASE_MAXCONNIDLETIME" default:"5m"`  // макс. "время простоя" соединения
	}

	ExternalAPIConfig struct {
		LocationIQapiKey string `env:"LOCATIONIQ_API_KEY"`
	}

	RabbitMQConfig struct {
		Host     string `env:"RABBITMQ_HOST" default:"localhost"`
		Port     string `env:"RABBITMQ_PORT" default:"5672"`
		User     string `env:"RABBITMQ_USER" default:"guest"`
		Password string `env:"RABBITMQ_PASSWORD" default:"guest"`
	}

	WebSocketConfig struct {
		Port string `env:"WEBSOCKET_PORT" default:"8080"`
	}

	ServicesConfig struct {
		RideService           string `env:"SERVICES_RIDE_SERVICE" default:"3000"`
		DriverLocationService string `env:"SERVICES_DRIVER_LOCATION_SERVICE" default:"3001"`
		AdminService          string `env:"SERVICES_ADMIN_SERVICE" default:"3004"`
		AuthService           string `env:"SERVICES_AUTH_SERVICE" default:"3005"`
	}

	Auth struct {
		AccessTokenTTL  time.Duration `env:"AUTH_ACCESS_TOKEN_TTL" default:"15m"`
		RefreshTokenTTL time.Duration `env:"AUTH_REFRESH_TOKEN_TTL" default:"168h"`
		JWTSecret       string        `env:"AUTH_JWT_SECRET" default:"supersecretkey"`
	}
)

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

func (c RabbitMQConfig) GetDSN() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/",
		c.User,
		c.Password,
		c.Host,
		c.Port,
	)
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
