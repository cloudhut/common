package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func TestBuildWatchedTLSConfig(t *testing.T) {
	log := zap.NewExample()

	t.Run("no update", func(t *testing.T) {
		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		port := l.Addr().(*net.TCPAddr).Port

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, client")
		})

		tlsServerConfig, err := BuildWatchedTLSConfig(log,
			"./testdata/certs/localhost.crt", "./testdata/certs/localhost.key",
			nil)
		require.NoError(t, err)

		ts := &http.Server{
			Handler:   handler,
			TLSConfig: tlsServerConfig,
		}

		go func() {
			ts.ServeTLS(l, "", "")
		}()

		timer1 := time.NewTimer(10 * time.Millisecond)
		<-timer1.C

		t.Cleanup(func() {
			ts.Shutdown(context.Background())
		})

		tlsConfig := newTLSConfig(t, "./testdata/certs/localhost.crt", "./testdata/certs/localhost.key")
		tlsConfig.InsecureSkipVerify = true
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}

		res, err := client.Get("https://localhost:" + strconv.Itoa(port) + "/")
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "HTTP/1.1", res.Proto)

		pc := res.TLS.PeerCertificates[0]
		dur := pc.NotAfter.Sub(pc.NotBefore)
		assert.Equal(t, 360*24.0, dur.Hours())
	})

	t.Run("update no channel", func(t *testing.T) {
		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		port := l.Addr().(*net.TCPAddr).Port

		// temp directory
		dname, err := os.MkdirTemp("", "certs")
		require.NoError(t, err)

		t.Cleanup(func() {
			os.RemoveAll(dname)
		})

		// copy first set of certs to temp dir
		certFile := filepath.Join(dname, "secure.crt")
		keyFile := filepath.Join(dname, "secure.key")

		copyFile(t, "./testdata/certs/localhost.crt", certFile)
		copyFile(t, "./testdata/certs/localhost.key", keyFile)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, client")
		})

		tlsServerConfig, err := BuildWatchedTLSConfig(log, certFile, keyFile, nil)
		require.NoError(t, err)

		ts := &http.Server{
			Handler:   handler,
			TLSConfig: tlsServerConfig,
		}

		go func() {
			ts.ServeTLS(l, "", "")
		}()

		timer1 := time.NewTimer(10 * time.Millisecond)
		<-timer1.C

		t.Cleanup(func() {
			ts.Shutdown(context.Background())
		})

		tlsConfig := newTLSConfig(t, "./testdata/certs/localhost.crt", "./testdata/certs/localhost.key")
		tlsConfig.InsecureSkipVerify = true
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}

		res, err := client.Get("https://localhost:" + strconv.Itoa(port) + "/")
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "HTTP/1.1", res.Proto)

		pc := res.TLS.PeerCertificates[0]
		dur := pc.NotAfter.Sub(pc.NotBefore)
		assert.Equal(t, 360*24.0, dur.Hours())

		// update certs

		timer1 = time.NewTimer(10 * time.Millisecond)
		<-timer1.C

		copyFile(t, "./testdata/certs/localhost2.crt", certFile)
		copyFile(t, "./testdata/certs/localhost2.key", keyFile)

		// allow for hot reload
		timer1 = time.NewTimer(200 * time.Millisecond)
		<-timer1.C

		// check
		res, err = client.Get("https://localhost:" + strconv.Itoa(port) + "/")
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "HTTP/1.1", res.Proto)

		pc = res.TLS.PeerCertificates[0]
		dur = pc.NotAfter.Sub(pc.NotBefore)
		assert.Equal(t, 350*24.0, dur.Hours())
	})

	t.Run("update with channel", func(t *testing.T) {
		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		port := l.Addr().(*net.TCPAddr).Port

		// temp directory
		dname, err := os.MkdirTemp("", "certs")
		require.NoError(t, err)

		t.Cleanup(func() {
			os.RemoveAll(dname)
		})

		// copy first set of certs to temp dir
		certFile := filepath.Join(dname, "secure.crt")
		keyFile := filepath.Join(dname, "secure.key")

		copyFile(t, "./testdata/certs/localhost.crt", certFile)
		copyFile(t, "./testdata/certs/localhost.key", keyFile)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, client")
		})

		signalCh := make(chan TLSReloadNotifySignal, 10)

		count := atomic.Int32{}
		go func() {
			update := <-signalCh
			assert.NotEmpty(t, update)
			count.Add(1)
		}()

		tlsServerConfig, err := BuildWatchedTLSConfig(log, certFile, keyFile, signalCh)
		require.NoError(t, err)

		ts := &http.Server{
			Handler:   handler,
			TLSConfig: tlsServerConfig,
		}

		go func() {
			ts.ServeTLS(l, "", "")
		}()

		timer1 := time.NewTimer(10 * time.Millisecond)
		<-timer1.C

		t.Cleanup(func() {
			ts.Shutdown(context.Background())
		})

		tlsConfig := newTLSConfig(t, "./testdata/certs/localhost.crt", "./testdata/certs/localhost.key")
		tlsConfig.InsecureSkipVerify = true
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}

		res, err := client.Get("https://localhost:" + strconv.Itoa(port) + "/")
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "HTTP/1.1", res.Proto)

		pc := res.TLS.PeerCertificates[0]
		dur := pc.NotAfter.Sub(pc.NotBefore)
		assert.Equal(t, 360*24.0, dur.Hours())

		// update certs

		timer1 = time.NewTimer(10 * time.Millisecond)
		<-timer1.C

		copyFile(t, "./testdata/certs/localhost2.crt", certFile)
		copyFile(t, "./testdata/certs/localhost2.key", keyFile)

		// allow for hot reload
		timer1 = time.NewTimer(200 * time.Millisecond)
		<-timer1.C

		// check
		res, err = client.Get("https://localhost:" + strconv.Itoa(port) + "/")
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "HTTP/1.1", res.Proto)

		pc = res.TLS.PeerCertificates[0]
		dur = pc.NotAfter.Sub(pc.NotBefore)
		assert.Equal(t, 350*24.0, dur.Hours())

		v := count.Load()

		assert.Equal(t, int32(1), v)

		close(signalCh)
	})
}

func newTLSConfig(t *testing.T, certFile, keyFile string) *tls.Config {
	t.Helper()

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)

	caCertPool, err := x509.SystemCertPool()
	require.NoError(t, err)

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()

	dat, err := os.ReadFile(src)
	require.NoError(t, err)

	err = os.WriteFile(dst, dat, 0644)
	require.NoError(t, err)
}
