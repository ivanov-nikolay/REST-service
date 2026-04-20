package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBConfig     DBConfig
	AppConfig    AppConfig
	LoggerConfig LoggerConfig
}

type DBConfig struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

type AppConfig struct {
	ServerPort string
}

type LoggerConfig struct {
	LogLevel string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: no .env file found, using system env: %v", err)
	}

	return &Config{
		DBConfig: DBConfig{
			DBHost:     getEnv("DB_HOST", "localhost"),
			DBPort:     getEnv("DB_PORT", "5432"),
			DBUser:     getEnv("DB_USER", "postgres"),
			DBPassword: getEnv("DB_PASSWORD", "12345"),
			DBName:     getEnv("DB_NAME", "subscriptions"),
		},
		AppConfig: AppConfig{
			ServerPort: getEnv("SERVER_PORT", "8080"),
		},
		LoggerConfig: LoggerConfig{
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
	}
}

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}

func (c *Config) GetDBConnString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBConfig.DBHost,
		c.DBConfig.DBPort,
		c.DBConfig.DBUser,
		c.DBConfig.DBPassword,
		c.DBConfig.DBName,
	)
}
