package rest

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
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
		},
		Logger: logger,
	}

	if cfg.TLS.Enabled {
		tlsCfg, err := buildServerTLSConfig(logger, cfg.TLS)
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
	listener, err := net.Listen("tcp", net.JoinHostPort(s.cfg.HTTPListenAddress, strconv.Itoa(s.cfg.HTTPListenPort)))
	if err != nil {
		return err
	}
	s.Logger.Info("Server listening on address", zap.String("address", listener.Addr().String()), zap.Int("port", s.cfg.HTTPListenPort))

	err = s.Server.Serve(listener)
	if err != http.ErrServerClosed {
		return err
	}

	wg.Wait()
	s.Logger.Info("Stopped HTTP server")

	return nil
}

func buildServerTLSConfig(logger *zap.Logger, cfg TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFilepath, cfg.KeyFilepath)
	if err != nil {
		return nil, fmt.Errorf("failed loading TLS cert: %w", err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to setup file watcher for hot reloading tls certificates: %w", err)
	}

	var lock sync.RWMutex
	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Ignore all events that are neither remove nor write
				isRelevantEvent := event.Has(fsnotify.Remove) || event.Has(fsnotify.Write)
				if !isRelevantEvent {
					continue
				}

				if event.Has(fsnotify.Remove) {
					// Kubernetes uses symbolic links to create the illusion of atomic writes.
					// Thus, we have to watch for the remove event and reconfigure our watcher.
					// See: https://ahmet.im/blog/kubernetes-inotify/
					_ = watcher.Remove(event.Name)
					err = watcher.Add(cfg.CertFilepath)
					if err != nil {
						logger.Error("failed to re-add file watcher",
							zap.String("file_path", cfg.CertFilepath),
							zap.Error(err))
					}

					err = watcher.Add(cfg.KeyFilepath)
					if err != nil {
						logger.Warn("failed to re-add file watcher",
							zap.String("file_path", cfg.KeyFilepath),
							zap.Error(err))
					}
				}

				logger.Info("key or certificate file has changed. hot reloading the tls certificate")

				newCert, err := tls.LoadX509KeyPair(cfg.CertFilepath, cfg.KeyFilepath)
				if err != nil {
					logger.Error("failed to load certificates", zap.Error(err))
					continue
				}
				lock.Lock()
				cert = newCert
				lock.Unlock()
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("tls certificate watcher error", zap.Error(err))
			case <-signalCh:
				return
			}
		}
	}()
	if err := watcher.Add(cfg.CertFilepath); err != nil {
		return nil, fmt.Errorf("failed to setup watcher for cert file: %w", err)
	}
	if err = watcher.Add(cfg.KeyFilepath); err != nil {
		return nil, fmt.Errorf("failed to setup watcher for key file: %w", err)
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			lock.RLock()
			defer lock.RUnlock()

			return &cert, nil
		},
	}, nil
}
