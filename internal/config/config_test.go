package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	os.Setenv("DB_HOST", "testhost")
	defer os.Unsetenv("DB_HOST")
	cfg := Load()
	assert.Equal(t, "testhost", cfg.DBConfig.DBHost)
	assert.Equal(t, "5432", cfg.DBConfig.DBPort)
}

func TestGetDBConnString(t *testing.T) {
	cfg := &Config{
		DBConfig: DBConfig{
			DBHost:     "localhost",
			DBPort:     "5432",
			DBUser:     "user",
			DBPassword: "password",
			DBName:     "db",
		},
	}
	connStr := cfg.GetDBConnString()
	assert.Contains(t, connStr, "host=localhost")
	assert.Contains(t, connStr, "port=5432")
	assert.Contains(t, connStr, "user=user")
	assert.Contains(t, connStr, "password=password")
	assert.Contains(t, connStr, "dbname=db")
}
