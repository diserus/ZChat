package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Redis    RedisConfig
	Log      LogConfig
}

type ServerConfig struct {
	Host string `env:"SERVER_HOST" envDefault:"localhost"`
	Port string `env:"SERVER_PORT" envDefault:"8080"`
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST,required"`
	Port     string `env:"DB_PORT"         envDefault:"5432"`
	User     string `env:"DB_USER,required"`
	Password string `env:"DB_PASSWORD,required"`
	Name     string `env:"DB_NAME,required"`
	SSLMode  string `env:"DB_SSLMODE"      envDefault:"disable"`
}

type JWTConfig struct {
	AccessSecret  string        `env:"JWT_ACCESS_SECRET,required"`
	RefreshSecret string        `env:"JWT_REFRESH_SECRET,required"`
	AccessTTL     time.Duration `env:"JWT_ACCESS_TTL"  envDefault:"15m"`
	RefreshTTL    time.Duration `env:"JWT_REFRESH_TTL" envDefault:"168h"`
}

type RedisConfig struct {
	Host        string        `env:"REDIS_HOST"           envDefault:"localhost"`
	Port        string        `env:"REDIS_PORT"           envDefault:"6379"`
	Password    string        `env:"REDIS_PASSWORD"       envDefault:""`
	DB          int           `env:"REDIS_DB"             envDefault:"0"`
	PresenceTTL time.Duration `env:"REDIS_PRESENCE_TTL"   envDefault:"2m"`
	OfflineTTL  time.Duration `env:"REDIS_OFFLINE_TTL"    envDefault:"168h"`
}

type LogConfig struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"INFO"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return cfg, nil
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}
