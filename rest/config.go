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

	BasePath                        string `yaml:"basePath"`
	SetBasePathFromXForwardedPrefix bool   `yaml:"setBasePathFromXForwardedPrefix"`
	StripPrefix                     bool   `yaml:"stripPrefix"`
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

	f.StringVar(&c.BasePath, "server.base-path", "", "Sets the subpath (root prefix) under which Kowl is reachable. If you want to host Kowl under 'your.domain.com/kowl/' you'd set the base path to 'kowl/'. The default is an empty string which makes Kowl reachable under just 'domain.com/'. When using this setting (or letting the 'X-Forwarded-Prefix' header set it for you) remember to either leave 'strip-prefix' enabled, or use a proxy that can strip the base-path/prefix before it reaches Kowl.")
	f.BoolVar(&c.SetBasePathFromXForwardedPrefix, "server.set-base-path-from-x-forwarded-prefix", true, "When set to true, Kowl will use the 'X-Forwarded-Prefix' header as the base path. (When enabled the 'base-path' setting won't be used)")
	f.BoolVar(&c.StripPrefix, "server.strip-prefix", true, "If a base-path is set (either by the 'base-path' setting, or by the 'X-Forwarded-Prefix' header), they will be removed from the request url. You probably want to leave this enabled, unless you are using a proxy that can remove the prefix automatically (like Traefik's 'StripPrefix' option)")
}

func (c *Config) SetDefaults() {
	c.ServerGracefulShutdownTimeout = 30 * time.Second

	c.HTTPListenPort = 8080
	c.HTTPServerIdleTimeout = 30 * time.Second
	c.HTTPServerReadTimeout = 30 * time.Second
	c.HTTPServerWriteTimeout = 30 * time.Second

	c.CompressionLevel = 4

	c.BasePath = ""
	c.SetBasePathFromXForwardedPrefix = true
	c.StripPrefix = true
}
