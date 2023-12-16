package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/grepplabs/backstream/handler"
	"github.com/grepplabs/backstream/ws"
)

func main() {
	addr := flag.String("addr", ":8080", "Listen address")
	logLevel := flag.String("log-level", "debug", "Log level")
	flag.Parse()

	codec := handler.NewHttpProtoCodec()
	proxyHandler := handler.NewProxyHandler(codec, handler.WithProxyDefaultRequestTimeout(3*time.Second))
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: getLevel(*logLevel)}))
	serve := ws.NewServe(context.Background(), proxyHandler, codec.MessageCodec(), ws.WithServeLogger(logger), ws.WithRequireClientId(true))

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serve.HandleWS(w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serve.HandleProxy(w, r)
	})
	server := &http.Server{
		Addr:              *addr,
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           mux,
	}
	logger.Info("starting proxy on " + *addr)
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
