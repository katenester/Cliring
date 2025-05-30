package config

import (
	"github.com/caarlos0/env/v6"
	"github.com/sirupsen/logrus"
)

type Config struct {
	HTTPPort string `env:"HTTP_PORT" envDefault:"8080"`
	Postgres Postgres
}

type Postgres struct {
	DSN                   string `env:"DSN" envDefault:"postgres://postgres:hFAClzgcwH5QNmEja8CdzwVDMCnxxm@localhost:5440/cliring?sslmode=disable"`
	MigrationsDir         string `env:"MIGRATION_MIGRATIONS_DIR" envDefault:"migrations"`
	MigrationVersionTable string `env:"MIGRATION_VERSION_TABLE" envDefault:"schema_version"`
}

func New() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		logrus.Fatal(err)
		return nil, err
	}
	return cfg, nil
}
