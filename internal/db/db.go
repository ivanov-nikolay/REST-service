package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ivanov-nikolay/REST-service/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sirupsen/logrus"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type LogrusMigrateLogger struct {
	log *logrus.Logger
}

func (l *LogrusMigrateLogger) Printf(format string, v ...interface{}) {
	l.log.Infof(format, v...)
}

func (l *LogrusMigrateLogger) Verbose() bool {
	return true
}

func NewGormDB(cfg *config.Config) (*gorm.DB, error) {
	gormConfig := &gorm.Config{}
	if cfg.LoggerConfig.LogLevel != "debug" {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(pg.Open(cfg.GetDBConnString()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to databaseL %w", err)
	}

	return db, nil
}

func RunMigrations(cfg *config.Config, log *logrus.Logger) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory %w", err)
	}

	migrationsPath := filepath.Join(wd, "migrations")
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist %s", migrationsPath)
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBConfig.DBUser,
		cfg.DBConfig.DBPassword,
		cfg.DBConfig.DBHost,
		cfg.DBConfig.DBPort,
		cfg.DBConfig.DBName,
	)

	log.WithFields(logrus.Fields{
		"source": sourceURL,
		"target": connStr,
	}).Info("migrations configuration")

	m, err := migrate.New(sourceURL, connStr)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance %w", err)
	}
	defer m.Close()

	m.Log = &LogrusMigrateLogger{
		log: log,
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		log.WithError(err).Warn("could not get migration version")
	} else if err == migrate.ErrNilVersion {
		log.Info("no migrations applied yet - fresh database")
	} else {
		log.WithFields(logrus.Fields{
			"version": version,
			"dirty":   dirty,
		}).Info("current migration state")
	}

	if dirty {
		log.WithField("version", version).Warn("database is dirty, attempting to fix...")

		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.WithError(err).Warn("down migration failed, forcing version...")
			if err := m.Force(int(version)); err != nil {

				return fmt.Errorf("failed to force version %d: %w", version, err)
			}
			log.WithField("version", version).Info("successfully forced clean version")
		} else {
			log.Info("successfully rolled back from dirty state\n")
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		if strings.Contains(err.Error(), "no migration found for version 0") {
			log.Info("database is fresh, no migrations needed")
			return nil
		}

		return fmt.Errorf("failed to run migrations: %w", err)
	}

	finalVersion, finalDirty, _ := m.Version()
	log.WithFields(logrus.Fields{
		"version": finalVersion,
		"dirty":   finalDirty,
	}).Info("migrations applied successfully")

	return nil
}
