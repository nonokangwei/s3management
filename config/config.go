package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
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
func Load() (*Config, error) {
	cfg := &Config{}

	// Defaults from environment variables
	var errs []string

	cfg.ListenAddr = getEnv("LISTEN_ADDR", ":8080")
	cfg.GCSProjectID = strings.TrimSpace(getEnv("GCS_PROJECT_ID", ""))
	cfg.LogLevel = strings.ToLower(getEnv("LOG_LEVEL", "info"))

	readTimeout, err := getDurationEnv("READ_TIMEOUT", 30*time.Second)
	if err != nil {
		errs = append(errs, fmt.Sprintf("READ_TIMEOUT: %v", err))
	} else {
		cfg.ReadTimeout = readTimeout
	}

	writeTimeout, err := getDurationEnv("WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		errs = append(errs, fmt.Sprintf("WRITE_TIMEOUT: %v", err))
	} else {
		cfg.WriteTimeout = writeTimeout
	}

	idleTimeout, err := getDurationEnv("IDLE_TIMEOUT", 120*time.Second)
	if err != nil {
		errs = append(errs, fmt.Sprintf("IDLE_TIMEOUT: %v", err))
	} else {
		cfg.IdleTimeout = idleTimeout
	}

	shutdownTimeout, err := getDurationEnv("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		errs = append(errs, fmt.Sprintf("SHUTDOWN_TIMEOUT: %v", err))
	} else {
		cfg.ShutdownTimeout = shutdownTimeout
	}

	gcsTimeout, err := getDurationEnv("GCS_REQUEST_TIMEOUT", 30*time.Second)
	if err != nil {
		errs = append(errs, fmt.Sprintf("GCS_REQUEST_TIMEOUT: %v", err))
	} else if gcsTimeout <= 0 {
		errs = append(errs, "GCS_REQUEST_TIMEOUT must be greater than 0")
	} else {
		cfg.GCSRequestTimeout = gcsTimeout
	}

	maxBody, err := getInt64Env("MAX_REQUEST_BODY_KB", 256)
	if err != nil {
		errs = append(errs, fmt.Sprintf("MAX_REQUEST_BODY_KB: %v", err))
	} else if maxBody <= 0 {
		errs = append(errs, "MAX_REQUEST_BODY_KB must be greater than 0")
	} else {
		cfg.MaxRequestBodyKB = maxBody
	}

	// Command-line flags override environment variables
	flag.StringVar(&cfg.ListenAddr, "addr", cfg.ListenAddr, "Listen address (host:port)")
	flag.StringVar(&cfg.GCSProjectID, "project", cfg.GCSProjectID, "GCS project ID (required)")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level: debug, info, warn, error")
	flag.Parse()

	if cfg.GCSProjectID == "" {
		errs = append(errs, "GCS project ID is required. Set GCS_PROJECT_ID env var or use --project flag")
	}

	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "; "))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d, nil
		}
		return 0, fmt.Errorf("invalid duration %q", v)
	}
	return fallback, nil
}

func getInt64Env(key string, fallback int64) (int64, error) {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n, nil
		}
		return 0, fmt.Errorf("invalid integer %q", v)
	}
	return fallback, nil
}
