package logging

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"
)

// Config for an slog logger
type Config struct {
	LogLevelInput string `yaml:"level"`
	LogLevel      slog.Level
}

// RegisterFlags adds the flags required to config the server
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.Set("info")
	f.Var(cfg, "logging.level", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error]")
}

// String implements the flag.Value interface
func (cfg *Config) String() string {
	return cfg.LogLevelInput
}

// Set updates the value of the allowed log level by implementing the flag.Value interface
func (cfg *Config) Set(logLevel string) error {
	switch strings.ToLower(logLevel) {
	case "", "info":
		cfg.LogLevel = slog.LevelInfo
	case "debug":
		cfg.LogLevel = slog.LevelDebug
	case "warn":
		cfg.LogLevel = slog.LevelWarn
	case "error":
		cfg.LogLevel = slog.LevelError
	default:
		fmt.Printf("Invalid log level supplied: %q. Defaulting to info.", logLevel)
		cfg.LogLevel = slog.LevelInfo
	}
	cfg.LogLevelInput = logLevel

	return nil
}

func (c *Config) SetDefaults() {
	c.LogLevelInput = "info"
	c.LogLevel = slog.LevelInfo
}
