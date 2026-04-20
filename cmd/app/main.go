package main

import (
	"github.com/ivanov-nikolay/REST-service/internal/config"
	"github.com/ivanov-nikolay/REST-service/internal/db"

	"github.com/sirupsen/logrus"
)

func main() {
	cfg := config.Load()

	log := logrus.New()
	level, _ := logrus.ParseLevel(cfg.LoggerConfig.LogLevel)
	log.SetLevel(level)
	log.SetFormatter(&logrus.JSONFormatter{})

	if err := db.RunMigrations(cfg, log); err != nil {
		log.WithError(err).Fatal("Failed to run migrations")
	}

	gormDB, err := db.NewGormDB(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}

	_ = gormDB
	log.Println("Migrations allied successfully!")
}
