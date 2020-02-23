package rest

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config for a HTTP server
type Config struct {
	ServerGracefulShutdownTimeout time.Duration `yaml:"gracefulShutdownTimeout"`

	HTTPListenPort         int           `yaml:"listenPort"`
	HTTPServerReadTimeout  time.Duration `yaml:"readTimeout"`
	HTTPServerWriteTimeout time.Duration `yaml:"writeTimeout"`
	HTTPServerIdleTimeout  time.Duration `yaml:"idleTimeout"`

	CompressionLevel int `yaml:"compressionLevel"`
}

// RegisterFlags adds the flags required to config the server
func (c *Config) RegisterFlags(f *flag.FlagSet) {
	f.DurationVar(&c.ServerGracefulShutdownTimeout, "server.graceful-shutdown-timeout", 30*time.Second, "Timeout for graceful shutdowns")

	f.IntVar(&c.HTTPListenPort, "server.http.listen-port", 8080, "HTTP server listen port")
	// Get "PORT" environment variable because CloudRun tells us what Port to use
	portEnv := os.Getenv("PORT")
	if portEnv != "" {
		port, err := strconv.Atoi(portEnv)
		if err != nil {
			panic("failed to parse port environment variable")
		}
		c.HTTPListenPort = port
	}

	f.DurationVar(&c.HTTPServerReadTimeout, "server.http.read-timeout", 30*time.Second, "Read timeout for HTTP server")
	f.DurationVar(&c.HTTPServerWriteTimeout, "server.http.write-timeout", 30*time.Second, "Write timeout for HTTP server")
	f.DurationVar(&c.HTTPServerIdleTimeout, "server.http.idle-timeout", 120*time.Second, "Idle timeout for HTTP server")
	f.IntVar(&c.CompressionLevel, "server.compression-level", 4, "Compression level applied to all http responses. Valid values are: 0-9 (0=completely disable compression middleware, 1=weakest compression, 9=best compression)")
}

func (c *Config) SetDefaults() {
	c.ServerGracefulShutdownTimeout = 30 * time.Second

	c.HTTPListenPort = 8080
	c.HTTPServerIdleTimeout = 30 * time.Second
	c.HTTPServerReadTimeout = 30 * time.Second
	c.HTTPServerWriteTimeout = 30 * time.Second

	c.CompressionLevel = 4
}
