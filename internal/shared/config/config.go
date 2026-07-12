package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	S3       S3Config
	Rate     RateConfig
	CORS     CORSConfig
	AuthRate AuthRateConfig
	Authz    AuthzConfig
}

type ServerConfig struct {
	Port int    `env:"PORT" envDefault:"8080"`
	Env  string `env:"ENV" envDefault:"development"`
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     int    `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER" envDefault:"ecampus"`
	Password string `env:"DB_PASSWORD" envDefault:"ecampus_dev"`
	Name     string `env:"DB_NAME" envDefault:"ecampus"`
	SSLMode  string `env:"DB_SSLMODE" envDefault:"prefer"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

func (d DatabaseConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	URL string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
}

type JWTConfig struct {
	Secret     string        `env:"JWT_SECRET,required"`
	AccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"1h"`
	RefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"168h"`
}

type S3Config struct {
	Endpoint  string `env:"S3_ENDPOINT" envDefault:"http://localhost:9000"`
	Bucket    string `env:"S3_BUCKET" envDefault:"ecampus"`
	AccessKey string `env:"S3_ACCESS_KEY" envDefault:"minioadmin"`
	SecretKey string `env:"S3_SECRET_KEY" envDefault:"minioadmin"`
	UseSSL    bool   `env:"S3_USE_SSL" envDefault:"false"`
}

type RateConfig struct {
	Enabled bool `env:"RATE_LIMIT_ENABLED" envDefault:"true"`
	RPS     int  `env:"RATE_LIMIT_RPS" envDefault:"100"`
	Burst   int  `env:"RATE_LIMIT_BURST" envDefault:"20"`
}

type CORSConfig struct {
	AllowedOrigins string `env:"CORS_ORIGINS" envDefault:"http://localhost:5173"`
}

// Origins returns the allowed origins as a list.
func (c CORSConfig) Origins() []string {
	return strings.Split(c.AllowedOrigins, ",")
}

type AuthRateConfig struct {
	Enabled       bool `env:"AUTH_RATE_LIMIT_ENABLED" envDefault:"true"`
	MaxAttempts   int  `env:"AUTH_RATE_MAX_ATTEMPTS" envDefault:"5"`
	WindowSeconds int  `env:"AUTH_RATE_WINDOW_SECONDS" envDefault:"300"`
}

// AuthzConfig selects where authorization policies live. "static" serves the
// defaults compiled into the binary (no DB rows, no cache, no admin
// endpoints — a policy change is a deploy); "db" seeds those defaults into
// postgres once and lets super-admins tune them at runtime.
type AuthzConfig struct {
	PolicyMode string `env:"AUTHZ_POLICY_MODE" envDefault:"static"`
}

// PoliciesInDB reports whether the db policy mode is selected.
func (a AuthzConfig) PoliciesInDB() bool { return a.PolicyMode == "db" }

func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if e := cfg.Server.Env; e != "development" && e != "production" {
		return nil, fmt.Errorf("ENV: %q (must be \"development\" or \"production\")", e)
	}
	if m := cfg.Authz.PolicyMode; m != "static" && m != "db" {
		return nil, fmt.Errorf("AUTHZ_POLICY_MODE: %q (must be \"static\" or \"db\")", m)
	}
	return cfg, nil
}
