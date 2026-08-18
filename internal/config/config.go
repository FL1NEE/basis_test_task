package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort string

	MySQLDSN string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret    string
	JWTTTL       time.Duration
	TaskCacheTTL time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		MySQLDSN:      getEnv("MYSQL_DSN", "app:app@tcp(localhost:3306)/bazis?parseTime=true&multiStatements=true"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-me"),
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	cfg.RedisDB = redisDB

	jwtTTLMinutes, err := strconv.Atoi(getEnv("JWT_TTL_MINUTES", "60"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid JWT_TTL_MINUTES: %w", err)
	}
	cfg.JWTTTL = time.Duration(jwtTTLMinutes) * time.Minute

	taskCacheTTLSeconds, err := strconv.Atoi(getEnv("TASK_CACHE_TTL_SECONDS", "300"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid TASK_CACHE_TTL_SECONDS: %w", err)
	}
	cfg.TaskCacheTTL = time.Duration(taskCacheTTLSeconds) * time.Second

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}
