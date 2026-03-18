package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig           `mapstructure:"server"`
	Database DatabaseConfig         `mapstructure:"database"`
	Auth     AuthConfig             `mapstructure:"auth"`
	Media    MediaConfig            `mapstructure:"media"`
	Search   SearchConfig           `mapstructure:"search"`
	I18n     I18nConfig             `mapstructure:"i18n"`
	Security SecurityConfig         `mapstructure:"security"`
	Payment  PaymentConfig          `mapstructure:"payment"`
	Plugins  map[string]interface{} `mapstructure:"plugins"`
}

type PaymentConfig struct {
	EncryptionKey string `mapstructure:"encryption_key"`
}

type ServerConfig struct {
	Host          string        `mapstructure:"host"`
	Port          int           `mapstructure:"port"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	MaxBodySize   int64         `mapstructure:"max_body_size"`
	MaxUploadSize int64         `mapstructure:"max_upload_size"`
	CORS          CORSConfig    `mapstructure:"cors"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
}

type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type AuthConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

type MediaConfig struct {
	Storage   string      `mapstructure:"storage"`
	LocalPath string      `mapstructure:"local_path"`
	S3        S3Config    `mapstructure:"s3"`
}

type S3Config struct {
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

type SearchConfig struct {
	Engine string `mapstructure:"engine"`
}

type I18nConfig struct {
	DefaultLocale    string   `mapstructure:"default_locale"`
	AvailableLocales []string `mapstructure:"available_locales"`
}

type SecurityConfig struct {
	RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
	BcryptCost int              `mapstructure:"bcrypt_cost"`
	CSRF       CSRFConfig       `mapstructure:"csrf"`
	BruteForce BruteForceConfig `mapstructure:"brute_force"`
}

type BruteForceConfig struct {
	MaxAttempts  int           `mapstructure:"max_attempts"`
	LockDuration time.Duration `mapstructure:"lock_duration"`
}

type CSRFConfig struct {
	// Secure sets the Secure flag on the csrf_token cookie.
	// Enable in production when serving over HTTPS.
	Secure bool `mapstructure:"secure"`
}

type RateLimitConfig struct {
	RequestsPerMinute int                     `mapstructure:"requests_per_minute"`
	Burst             int                     `mapstructure:"burst"`
	Login             EndpointRateLimitConfig `mapstructure:"login"`
	Register          EndpointRateLimitConfig `mapstructure:"register"`
	Checkout          EndpointRateLimitConfig `mapstructure:"checkout"`
	GuestOrder        EndpointRateLimitConfig `mapstructure:"guest_order"`
}

type EndpointRateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.max_body_size", 1<<20)
	v.SetDefault("server.max_upload_size", 35<<20)
	v.SetDefault("server.cors.allowed_origins", []string{"http://localhost:5173"})
	v.SetDefault("server.cors.allowed_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE"})

	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")

	v.SetDefault("auth.access_token_ttl", "15m")
	v.SetDefault("auth.refresh_token_ttl", "168h")

	v.SetDefault("media.storage", "local")
	v.SetDefault("media.local_path", "./uploads")

	v.SetDefault("search.engine", "postgres")

	v.SetDefault("i18n.default_locale", "de-DE")
	v.SetDefault("i18n.available_locales", []string{"de-DE", "en-US"})

	v.SetDefault("security.rate_limit.requests_per_minute", 300)
	v.SetDefault("security.rate_limit.burst", 50)
	v.SetDefault("security.rate_limit.login.requests_per_minute", 10)
	v.SetDefault("security.rate_limit.register.requests_per_minute", 5)
	v.SetDefault("security.rate_limit.checkout.requests_per_minute", 10)
	v.SetDefault("security.rate_limit.guest_order.requests_per_minute", 10)
	v.SetDefault("security.bcrypt_cost", 12)
	v.SetDefault("security.brute_force.max_attempts", 5)
	v.SetDefault("security.brute_force.lock_duration", "60m")

	// Config file
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/stoa/")
	}

	// Environment variables
	v.SetEnvPrefix("STOA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
