package rest

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

// Server struct to handle a common http routing server
type redirectServer struct {
	cfg *Config

	Server *http.Server
	Logger *slog.Logger
}

// newRedirectServer creates a new server whose sole purpose it is to
// redirect HTTP requests to their equivalent HTTPS version.
func newRedirectServer(cfg *Config, logger *slog.Logger) *Server {
	copiedCfg := *cfg
	copiedCfg.TLS.Enabled = false

	redirectPort := cfg.HTTPSListenPort
	if cfg.AdvertisedHTTPSListenPort != 0 {
		redirectPort = cfg.AdvertisedHTTPSListenPort
	}
	return &Server{
		cfg: &copiedCfg,
		Server: &http.Server{
			ReadTimeout:  cfg.HTTPServerReadTimeout,
			WriteTimeout: cfg.HTTPServerWriteTimeout,
			IdleTimeout:  cfg.HTTPServerIdleTimeout,
			ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				host, _, _ := net.SplitHostPort(r.Host)
				u := r.URL
				u.Host = net.JoinHostPort(host, strconv.Itoa(redirectPort))
				u.Scheme = "https"
				w.Header().Set("Connection", "close")
				http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
			}),
		},
		Logger: logger,
	}
}
