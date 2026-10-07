package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config 从环境变量加载的全局配置。
type Config struct {
	AppEnv       string `env:"APP_ENV" envDefault:"development"`
	ServerPort   string `env:"SERVER_PORT" envDefault:"8080"`
	DBHost       string `env:"DB_HOST" envDefault:"127.0.0.1"`
	DBPort       string `env:"DB_PORT" envDefault:"3306"`
	DBUser       string `env:"DB_USER" envDefault:"contract"`
	DBPassword   string `env:"DB_PASSWORD"`
	DBName       string `env:"DB_NAME" envDefault:"contractapi"`
	JWTSecret    string `env:"JWT_SECRET"`
	JWTExpireHours int  `env:"JWT_EXPIRE_HOURS" envDefault:"24"`
}

// Load 解析环境变量并做基础校验。
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("load config: JWT_SECRET must not be empty")
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("load config: DB_PASSWORD must not be empty")
	}
	return cfg, nil
}

// DSN 返回 GORM MySQL 数据源。
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}
