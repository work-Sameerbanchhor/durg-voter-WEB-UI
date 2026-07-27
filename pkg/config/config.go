package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	DBPath             string
	Environment        string
	RateLimitRPS       int
	RateLimitBurst     int
	CORSAllowedOrigins string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "database/durg_voters.duckdb"
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "production"
	}

	rps := getEnvAsInt("RATE_LIMIT_RPS", 50)
	burst := getEnvAsInt("RATE_LIMIT_BURST", 100)
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins == "" {
		origins = "*"
	}

	return &Config{
		Port:               port,
		DBPath:             dbPath,
		Environment:        env,
		RateLimitRPS:       rps,
		RateLimitBurst:     burst,
		CORSAllowedOrigins: origins,
		ReadTimeout:        15 * time.Second,
		WriteTimeout       : 15 * time.Second,
		IdleTimeout:        60 * time.Second,
	}
}

func getEnvAsInt(key string, defaultVal int) int {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}
