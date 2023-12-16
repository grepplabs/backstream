package handler

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grepplabs/backstream/ws"
	"github.com/oklog/run"
	"github.com/stretchr/testify/require"
)

const (
	certDir = "../tests/cfssl/certs"
)

func TestHttpProxy(t *testing.T) {
	t.Skip("IT test")

	codec := NewHttpJsonCodec()

	proxyAddr := ":8080"
	httpAddr := ":8081"

	var group run.Group
	proxyPort, err := addProxyServer(context.Background(), &group, codec, proxyAddr)
	require.Nil(t, err)

	proxyAddr = net.JoinHostPort("0.0.0.0", strconv.Itoa(proxyPort))
	_, err = addHttpServer(context.Background(), &group, codec, httpAddr, proxyAddr)
	require.Nil(t, err)

	slog.Info("Run servers")
	slog.Info(fmt.Sprintf("#  curl localhost:%d/test", proxyPort))

	err = group.Run()
	slog.Info("Finished " + err.Error())
}

func TestHttpProxyTLS(t *testing.T) {
	t.Skip("IT test")

	codec := NewHttpJsonCodec()

	proxyAddr := ":9080"
	httpAddr := ":9081"

	var group run.Group
	proxyPort, err := addProxyServerTLS(context.Background(), &group, codec, proxyAddr)
	require.Nil(t, err)

	proxyAddr = net.JoinHostPort("0.0.0.0", strconv.Itoa(proxyPort))
	_, err = addHttpServerTLS(context.Background(), &group, codec, httpAddr, proxyAddr)
	require.Nil(t, err)

	slog.Info("Run servers")
	slog.Info(fmt.Sprintf("#  curl -k https://localhost:%d/test", proxyPort))

	err = group.Run()
	slog.Info("Finished " + err.Error())
}

func TestHttpProxyRequest(t *testing.T) {
	tests := []struct {
		name  string
		codec HttpCodec
		tls   bool
	}{
		{
			name:  "http json codec",
			codec: NewHttpJsonCodec(),
			tls:   false,
		},
		{
			name:  "http proto codec",
			codec: NewHttpProtoCodec(),
			tls:   false,
		},
		{
			name:  "https json codec",
			codec: NewHttpJsonCodec(),
			tls:   true,
		},
		{
			name:  "https proto codec",
			codec: NewHttpProtoCodec(),
			tls:   true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			proxyAddr := ":0"
			httpAddr := ":0"
			shutdownAfter := 5 * time.Second

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var group run.Group
			var (
				err       error
				proxyPort int
			)

			if tc.tls {
				proxyPort, err = addProxyServerTLS(ctx, &group, tc.codec, proxyAddr)
				require.Nil(t, err)
				proxyAddr = net.JoinHostPort("0.0.0.0", strconv.Itoa(proxyPort))
				_, err = addHttpServerTLS(ctx, &group, tc.codec, httpAddr, proxyAddr)
				require.Nil(t, err)
				addTestRequest(ctx, cancel, &group, "https", proxyAddr)
			} else {
				proxyPort, err = addProxyServer(ctx, &group, tc.codec, proxyAddr)
				require.Nil(t, err)
				proxyAddr = net.JoinHostPort("0.0.0.0", strconv.Itoa(proxyPort))
				_, err = addHttpServer(ctx, &group, tc.codec, httpAddr, proxyAddr)
				require.Nil(t, err)
				addTestRequest(ctx, cancel, &group, "http", proxyAddr)
			}
			addTestTimeout(ctx, cancel, &group, shutdownAfter)

			slog.Info("run servers")

			err = group.Run()
			require.NoError(t, err)
		})
	}
}

func testRequest(ctx context.Context, proto, proxyAddr string) error {
	u := url.URL{
		Scheme: proto,
		Host:   proxyAddr,
		Path:   "/test",
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	client := &http.Client{
		Transport: transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status OK, but got '%d': body: '%s'", resp.StatusCode, string(body))
	}
	if string(body) != "OK" {
		return fmt.Errorf("expected body '%s', but got '%s'", "OK", string(body))
	}
	return nil
}

func addTestTimeout(ctx context.Context, cancel context.CancelFunc, group *run.Group, shutdownAfter time.Duration) {
	group.Add(func() error {
		<-ctx.Done()
		return ctx.Err()
	}, func(err error) {
	})
	time.AfterFunc(shutdownAfter, func() {
		slog.Warn("Context cancelling")
		cancel()
	})
}

func addTestRequest(ctx context.Context, cancel context.CancelFunc, group *run.Group, proto string, proxyAddr string) {
	group.Add(func() error {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				err := testRequest(ctx, proto, proxyAddr)
				if err != nil {
					slog.Warn("test request failed", slog.String("error", err.Error()))
					continue
				} else {
					slog.Info("test request success")
					cancel()
					return nil
				}
			}
		}
	}, func(err error) {
	})
}

func addHttpServer(ctx context.Context, group *run.Group, codec HttpCodec, httpAddr string, proxyAddr string) (int, error) {
	server := newHttpServer(ctx, codec, httpAddr, proxyAddr, nil)
	return addServer(ctx, group, httpAddr, server, "http server", nil)
}

func addProxyServer(ctx context.Context, group *run.Group, codec HttpCodec, proxyAddr string) (int, error) {
	server := newProxyServer(ctx, codec, proxyAddr)
	return addServer(ctx, group, proxyAddr, server, "proxy server", nil)
}

func addHttpServerTLS(ctx context.Context, group *run.Group, codec HttpCodec, httpAddr string, proxyAddr string) (int, error) {
	caCert, err := os.ReadFile(path.Join(certDir, "ca.pem"))
	if err != nil {
		return 0, err
	}
	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(caCert)
	clientTLSConfig := &tls.Config{
		RootCAs: rootCAs,
	}

	server := newHttpServer(ctx, codec, httpAddr, proxyAddr, clientTLSConfig)
	return addServer(ctx, group, httpAddr, server, "http server tls", nil)
}

func addProxyServerTLS(ctx context.Context, group *run.Group, codec HttpCodec, proxyAddr string) (int, error) {
	server := newProxyServer(ctx, codec, proxyAddr)
	var err error
	serverTLSConfig := &tls.Config{}
	serverTLSConfig.Certificates = make([]tls.Certificate, 1)
	serverTLSConfig.Certificates[0], err = tls.LoadX509KeyPair(path.Join(certDir, "server.pem"), path.Join(certDir, "server-key.pem"))
	if err != nil {
		return 0, err
	}
	return addServer(ctx, group, proxyAddr, server, "proxy server tls", serverTLSConfig)
}

func addServer(ctx context.Context, group *run.Group, addr string, server *http.Server, name string, serverTLSConfig *tls.Config) (int, error) {
	// server.ListenAndServe() is not used, as we need to get the port if provided bind port was 0.
	var (
		ln  net.Listener
		err error
	)
	if serverTLSConfig != nil {
		ln, err = tls.Listen("tcp", addr, serverTLSConfig)
	} else {
		ln, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port

	group.Add(func() error {
		slog.Info("starting " + name + " " + ln.Addr().String())
		if err = server.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}, func(error) {
		if err = server.Shutdown(ctx); err != nil {
			slog.Warn(name+" shutdown error", slog.String("error", err.Error()))
		}
	})
	return port, nil
}

func newHttpServer(ctx context.Context, codec HttpCodec, httpAddr string, proxyAddr string, clientTLSConfig *tls.Config) *http.Server {
	scheme := "ws"
	if clientTLSConfig != nil {
		scheme = "wss"
	}
	u := url.URL{Scheme: scheme, Host: proxyAddr, Path: "/ws"}

	mux := http.NewServeMux()
	wsHandler := NewRecoveryHandler(NewHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
		// forward all WS events to http server
		mux.ServeHTTP(w, r)
	}, codec), slog.Default())
	client := ws.NewClient(ctx, u.String(), wsHandler, codec.MessageCodec(), ws.WithClientTLSConfig(clientTLSConfig))
	client.Start()

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		msg := "handle " + r.URL.Path
		if r.URL.RawQuery != "" {
			msg = msg + "?" + r.URL.RawQuery
		}
		slog.Info(msg)
		w.Header().Add("x-test", uuid.New().String())
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
	})
	server := &http.Server{
		Addr:              httpAddr,
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           mux,
	}
	return server
}

func newProxyServer(ctx context.Context, codec HttpCodec, proxyAddr string) *http.Server {
	handler := NewProxyHandler(codec)
	serve := ws.NewServe(ctx, handler, codec.MessageCodec(), ws.WithRequireClientId(false))

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serve.HandleWS(w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serve.HandleProxy(w, r)
	})
	server := &http.Server{
		Addr:              proxyAddr,
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           mux,
	}
	return server
}
