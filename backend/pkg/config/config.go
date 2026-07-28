package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	DBPath             string
	Environment        string
	RateLimitRPS       int
	RateLimitBurst     int
	CORSAllowedOrigins string
	GeminiAPIKey       string
	GeminiModel        string
	SecretHeader       string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
}

func LoadConfig() *Config {
	loadDotEnv(".env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "7860"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		if _, err := os.Stat("backend/database/durg_voters.duckdb"); err == nil {
			dbPath = "backend/database/durg_voters.duckdb"
		} else {
			dbPath = "database/durg_voters.duckdb"
		}
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "production"
	}

	rps := getEnvAsInt("RATE_LIMIT_RPS", 15)
	burst := getEnvAsInt("RATE_LIMIT_BURST", 20)
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins == "" {
		origins = "*"
	}

	geminiKey := os.Getenv("GEMINI_API_KEY")
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-3.5-flash-lite"
	}

	secretHeader := os.Getenv("SECRET_HEADER")
	if secretHeader == "" {
		secretHeader = os.Getenv("SECRET_HEADER_VALUE")
	}
	if secretHeader == "" {
		secretHeader = "Sam2002@ABCD1234"
	}

	return &Config{
		Port:               port,
		DBPath:             dbPath,
		Environment:        env,
		RateLimitRPS:       rps,
		RateLimitBurst:     burst,
		CORSAllowedOrigins: origins,
		GeminiAPIKey:       geminiKey,
		GeminiModel:        geminiModel,
		SecretHeader:       secretHeader,
		ReadTimeout:        15 * time.Second,
		WriteTimeout:       15 * time.Second,
		IdleTimeout:        60 * time.Second,
	}
}

func loadDotEnv(filepath string) {
	paths := []string{filepath, "../" + filepath, "../../" + filepath}
	var file *os.File
	var err error
	for _, p := range paths {
		file, err = os.Open(p)
		if err == nil {
			break
		}
	}
	if file == nil || err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
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
