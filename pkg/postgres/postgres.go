package postgres

import (
	"cliring/config"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
	"github.com/sirupsen/logrus"
	"os"
)

var (
	ErrDSNRequired = errors.New("dsn required")
)

type Postgres struct {
	Conn   *pgx.Conn
	config config.Postgres
}

// New возвращает новый экземпляр Postgres, связанный с заданным именем источника данных.
func New(cfg *config.Config) *Postgres {
	db := &Postgres{
		Conn:   nil,
		config: cfg.Postgres,
	}
	return db
}

// Open открывает соединение с postgres.
func (db *Postgres) Open(ctx context.Context) (err error) {
	// Проверка, что задан DSN, прежде чем пытаться открыть соединение.
	if db.config.DSN == "" {
		return ErrDSNRequired
	}

	// Подключение соединения
	db.Conn, err = pgx.Connect(ctx, db.config.DSN)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	// Старт миграции
	logrus.Info("Starting database migration")
	if err := db.migrate(ctx); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	logrus.Info("Database migration completed successfully")
	return nil
}

// migrate- применяет миграции к базе данных с использованием tern.
func (db *Postgres) migrate(ctx context.Context) error {
	// Создаем мигрант tern
	migrator, err := migrate.NewMigrator(ctx, db.Conn, db.config.MigrationVersionTable)
	if err != nil {
		return fmt.Errorf("unable to initialize migrator: %w", err)
	}

	// Указываем директорию с миграциями

	err = migrator.LoadMigrations(os.DirFS(db.config.MigrationsDir))
	if err != nil {
		return fmt.Errorf("unable to load migrations from %s: %w", db.config.MigrationsDir, err)
	}

	// Применяем миграции
	err = migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("unable to apply migrations: %w", err)
	}

	return nil
}

// Close закрывает соединение с базой данных.
func (db *Postgres) Close(ctx context.Context) error {
	if db.Conn != nil {
		err := db.Conn.Close(ctx)
		if err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
		db.Conn = nil
	}
	return nil
}
