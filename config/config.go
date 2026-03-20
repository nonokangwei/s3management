package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config holds the proxy server configuration.
type Config struct {
	ListenAddr        string        // Host:port to bind the HTTP server
	GCSProjectID      string        // Google Cloud project ID for GCS operations
	LogLevel          string        // Logging verbosity: debug, info, warn, error
	ReadTimeout       time.Duration // Max duration for reading the entire request
	WriteTimeout      time.Duration // Max duration before timing out writes of the response
	IdleTimeout       time.Duration // Max time to wait for the next request on keep-alive connections
	ShutdownTimeout   time.Duration // Max time to wait for in-flight requests during shutdown
	GCSRequestTimeout time.Duration // Timeout for individual GCS API calls
	MaxRequestBodyKB  int64         // Max request body size in KB
}

// Load reads configuration from environment variables and command-line flags.
// Flags take precedence over environment variables.
func Load() *Config {
	cfg := &Config{}

	// Defaults from environment variables
	cfg.ListenAddr = getEnv("LISTEN_ADDR", ":8080")
	cfg.GCSProjectID = getEnv("GCS_PROJECT_ID", "")
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.ReadTimeout = getDurationEnv("READ_TIMEOUT", 30*time.Second)
	cfg.WriteTimeout = getDurationEnv("WRITE_TIMEOUT", 30*time.Second)
	cfg.IdleTimeout = getDurationEnv("IDLE_TIMEOUT", 120*time.Second)
	cfg.ShutdownTimeout = getDurationEnv("SHUTDOWN_TIMEOUT", 15*time.Second)
	cfg.GCSRequestTimeout = getDurationEnv("GCS_REQUEST_TIMEOUT", 30*time.Second)
	cfg.MaxRequestBodyKB = getInt64Env("MAX_REQUEST_BODY_KB", 256) // 256KB default, S3 CORS config max is 64KB

	// Command-line flags override environment variables
	flag.StringVar(&cfg.ListenAddr, "addr", cfg.ListenAddr, "Listen address (host:port)")
	flag.StringVar(&cfg.GCSProjectID, "project", cfg.GCSProjectID, "GCS project ID (required)")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level: debug, info, warn, error")
	flag.Parse()

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func getInt64Env(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return fallback
}
