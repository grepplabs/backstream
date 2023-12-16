package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/grepplabs/backstream/handler"
	"github.com/grepplabs/backstream/ws"
)

func main() {
	addr := flag.String("addr", ":8081", "Listen address")
	proxyUrl := flag.String("proxy-url", "ws://localhost:8080/ws", "Proxy websocket endpoint")
	clientID := flag.String("client-id", "4711", "Client ID")
	logLevel := flag.String("log-level", "debug", "Log level")
	disableWS := flag.Bool("disable-ws", false, "Disable websocket client")

	flag.Parse()

	mux := http.NewServeMux()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: getLevel(*logLevel)})).With(slog.String("client-id", *clientID))

	if !*disableWS {
		codec := handler.NewHttpProtoCodec()
		wsHandler := handler.NewRecoveryHandler(handler.NewHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
			mux.ServeHTTP(w, r)
		}, codec, handler.WithHTTPDefaultRequestTimeout(3*time.Second)), logger)
		client := ws.NewClient(context.Background(), *proxyUrl, wsHandler, codec.MessageCodec(), ws.WithClientLogger(logger), ws.WithClientID(*clientID))
		client.Start()
	}
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("test request received")
		w.WriteHeader(200)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxyHttpBin(w, r)
	})
	server := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}
	logger.Info("starting http on " + *addr)
	if err := server.ListenAndServe(); err != nil {
		logger.Error(err.Error())
	}
}

func getLevel(s string) slog.Level {
	var level slog.Level
	err := level.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return level
}

func proxyHttpBin(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	req, err := toHttpBinRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
}

func toHttpBinRequest(r *http.Request) (*http.Request, error) {
	u := r.URL
	u.Scheme = "http"
	u.Host = "httpbin.org"
	request, err := http.NewRequestWithContext(r.Context(), r.Method, u.String(), r.Body)
	if err != nil {
		return nil, err
	}
	request.Header = r.Header
	return request, nil
}
