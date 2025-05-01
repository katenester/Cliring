package postgres_test

import (
	pg "cliring-diagram/pkg/postgres"
	"context"
	"fmt"
	"mermaid-diagram/internal/config"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestNew(t *testing.T) {
	type args struct {
		cfg config.Config
		log zerolog.Logger
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "",
			args: args{
				cfg: config.Config{
					Postgres: config.Postgres{
						DSN:                   "dsn",
						Cert:                  "cer",
						MigrationsDir:         "dir",
						MigrationConfig:       "cfg",
						MigrationVersion:      "1",
						MigrationVersionTable: "scheme",
						MigrationAuto:         false,
					},
					Sentry:   nil,
					LogLevel: "info",
					IP:       "0.0.0.0",
					HTTPPort: "80",
				},
				log: zerolog.Nop(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, pg.New(&tt.args.cfg, tt.args.log))
		})
	}
}

func TestDB_OpenErr(t *testing.T) {
	ctx := context.Background()

	db := pg.New(&config.Config{}, zerolog.Nop())
	require.Error(t, db.Open(ctx))

	db = pg.New(&config.Config{Postgres: config.Postgres{DSN: "dsn"}}, zerolog.Nop())
	require.Error(t, db.Open(ctx))

	db = pg.New(&config.Config{Postgres: config.Postgres{DSN: "user=postgres password=pass host=localhost port=7432 dbname=orders sslmode=disable"}}, zerolog.Nop())
	require.Error(t, db.Open(ctx))
}

func TestDB_Open(t *testing.T) {
	ctx := context.Background()
	log := zerolog.Nop() // Отключение логирования

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16",
		ExposedPorts: []string{"7432:5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "pass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
	}
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	port := "7432"
	dsn := fmt.Sprintf("host=%s port=%s user=postgres password=pass dbname=testdb sslmode=disable", host, port)

	migrationsDir, err := filepath.Abs("./migration_test")
	require.NoError(t, err)

	migrationConfig, err := filepath.Abs("./migration_test/tern.conf")
	require.NoError(t, err)

	db := pg.New(&config.Config{
		Postgres: config.Postgres{
			DSN:                   dsn,
			Cert:                  "",
			MigrationsDir:         migrationsDir,
			MigrationConfig:       migrationConfig,
			MigrationVersion:      "last",
			MigrationVersionTable: "public.schema_version",
			MigrationAuto:         true,
		},
		Sentry:   nil,
		LogLevel: "debug",
		IP:       "0.0.0.0",
		HTTPPort: "80",
	}, log)

	require.NoError(t, db.Open(ctx))

	var q int
	require.NoError(t, db.Pool().QueryRow(ctx, "SELECT * from public.schema_version").Scan(&q))

	var userID, firstname string
	err = db.Pool().QueryRow(ctx, `SELECT id, firstname FROM users WHERE id = '123e4567-e89b-12d3-a456-426614174000'`).Scan(&userID, &firstname)
	require.NoError(t, err)
	require.Equal(t, "123e4567-e89b-12d3-a456-426614174000", userID)
	require.Equal(t, "John", firstname)

	require.NoError(t, db.Close())
}
