package tls

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// BuildWatchedTLSConfig builds TLS with integrated watched to reload certificate files when they are changed.
func BuildWatchedTLSConfig(logger *zap.Logger, certFile, keyFile string, notify chan<- *tls.Certificate) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
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
					err = watcher.Add(certFile)
					if err != nil {
						logger.Error("failed to re-add file watcher",
							zap.String("file_path", certFile),
							zap.Error(err))
					}

					err = watcher.Add(keyFile)
					if err != nil {
						logger.Warn("failed to re-add file watcher",
							zap.String("file_path", keyFile),
							zap.Error(err))
					}
				}

				logger.Info("hot reloading the TLS certificate")

				newCert, err := tls.LoadX509KeyPair(certFile, keyFile)
				if err != nil {
					logger.Error("failed to load certificates", zap.Error(err))
					continue
				}

				lock.Lock()
				cert = newCert
				lock.Unlock()

				if notify != nil {
					notify <- &newCert
				}

				logger.Info("successfully hot reloaded the TLS certificate")
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
	if err := watcher.Add(certFile); err != nil {
		return nil, fmt.Errorf("failed to setup watcher for cert file: %w", err)
	}
	if err = watcher.Add(keyFile); err != nil {
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
