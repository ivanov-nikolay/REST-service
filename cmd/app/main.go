package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "github.com/ivanov-nikolay/REST-service/docs"
	"github.com/ivanov-nikolay/REST-service/internal/config"
	"github.com/ivanov-nikolay/REST-service/internal/db"
	"github.com/ivanov-nikolay/REST-service/internal/handlers"
	"github.com/ivanov-nikolay/REST-service/internal/middleware"
	"github.com/ivanov-nikolay/REST-service/internal/repository"
	"github.com/ivanov-nikolay/REST-service/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Subscription Service API
// @version 1.0
// @description REST API for managing user subscriptions
// @host localhost:8080
// @BasePath /
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

	repo := repository.NewSubscriptionRepo(gormDB, log)
	svc := service.NewSubscriptionService(repo, log)
	handler := handlers.NewSubscriptionHandler(svc, log)

	e := echo.New()

	e.Use(middleware.RecoverWithLogger(log))
	e.Use(middleware.RequestLogger(log))

	e.Validator = &CustomValidator{validator: validator.New()}

	e.POST("/subscriptions", handler.Create)
	e.GET("/subscriptions/:id", handler.GetByID)
	e.PUT("/subscriptions/:id", handler.Update)
	e.DELETE("/subscriptions/:id", handler.Delete)
	e.GET("/subscriptions", handler.List)
	e.GET("/subscriptions/total-cost", handler.GetTotalCost)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	go func() {
		log.Infof("Starting server on port %s", cfg.AppConfig.ServerPort)
		if err := e.Start(":" + cfg.AppConfig.ServerPort); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("Server shutdown failed")
	}
	log.Info("Server exited gracefully")
}

type CustomValidator struct {
	validator *validator.Validate
}

func (v *CustomValidator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}
