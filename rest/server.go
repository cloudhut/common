package rest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/cloudhut/common/tls"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Server struct to handle a common http routing server
type Server struct {
	cfg *Config

	Router *chi.Mux
	Server *http.Server
	Logger *zap.Logger
}

// NewServer create server instance
func NewServer(cfg *Config, logger *zap.Logger, router *chi.Mux) (*Server, error) {
	server := &Server{
		cfg:    cfg,
		Router: router,
		Server: &http.Server{
			ReadTimeout:  cfg.HTTPServerReadTimeout,
			WriteTimeout: cfg.HTTPServerWriteTimeout,
			IdleTimeout:  cfg.HTTPServerIdleTimeout,
			Handler:      router,
			ErrorLog:     zap.NewStdLog(logger.Named("http_server")),
		},
		Logger: logger,
	}

	if cfg.TLS.Enabled {
		tlsCfg, err := tls.BuildWatchedTLSConfig(logger, cfg.TLS.CertFilepath, cfg.TLS.KeyFilepath, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		server.Server.TLSConfig = tlsCfg
	}

	return server, nil
}

// Start the HTTP server and blocks until we either receive a signal or the HTTP server returns an error.
func (s *Server) Start() error {
	var wg sync.WaitGroup
	wg.Add(1)

	// Listen for signals - shutdown the server if we receive one
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ServerGracefulShutdownTimeout)
		defer cancel()

		s.Logger.Info("Stopping HTTP server", zap.String("reason", "received signal"))
		s.Server.SetKeepAlivesEnabled(false)
		err := s.Server.Shutdown(ctx)
		if err != nil {
			s.Logger.Panic(err.Error())
		}

		wg.Done()
	}()

	// Serve HTTP server
	listenerPort := s.cfg.HTTPListenPort
	if s.cfg.TLS.Enabled {
		listenerPort = s.cfg.HTTPSListenPort
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(s.cfg.HTTPListenAddress, strconv.Itoa(listenerPort)))
	if err != nil {
		return err
	}
	s.Logger.Info("Server listening on address", zap.String("address", listener.Addr().String()), zap.Int("port", listenerPort))

	if s.cfg.TLS.Enabled {
		rdSrv := newRedirectServer(s.cfg, s.Logger)
		go func() {
			err := rdSrv.Start()
			if err != nil {
				s.Logger.Error("failed to start HTTP to HTTPS redirect server", zap.Error(err))
			}
		}()

		err = s.Server.ServeTLS(listener, "", "")
	} else {
		err = s.Server.Serve(listener)
	}
	if err != http.ErrServerClosed {
		return err
	}

	wg.Wait()
	s.Logger.Info("Stopped HTTP server", zap.String("address", listener.Addr().String()), zap.Int("port", listenerPort))

	return nil
}
