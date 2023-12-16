package ws

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/grepplabs/backstream/internal/message"
)

const HeaderClientId = "x-backstream-client-id"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Serve struct {
	pool            *Pool
	parent          context.Context
	handler         ProxyHandler
	logger          *slog.Logger
	requireClientId bool
	codec           Codec[*message.Message]
}

type ServeOption func(*Serve)

func WithServeLogger(logger *slog.Logger) ServeOption {
	return func(s *Serve) {
		s.logger = logger
	}
}

func WithRequireClientId(b bool) ServeOption {
	return func(s *Serve) {
		s.requireClientId = b
	}
}

type ProxyHandler interface {
	EventHandler
	ProxyRequest(conn *Conn, w http.ResponseWriter, r *http.Request) error
}

func NewServe(parent context.Context, handler ProxyHandler, codec Codec[*message.Message], opts ...ServeOption) *Serve {
	serve := &Serve{
		pool:            NewPool(),
		parent:          parent,
		handler:         handler,
		codec:           codec,
		logger:          slog.Default(),
		requireClientId: true,
	}
	for _, opt := range opts {
		opt(serve)
	}
	return serve
}

func (s *Serve) GetConnByID(id string) *Conn {
	return s.pool.GetConnByID(id)
}

func (s *Serve) GetConnsByID(id string) []*Conn {
	return s.pool.GetConnsByID(id)
}

func (s *Serve) HandleWS(w http.ResponseWriter, r *http.Request) {
	clientID := r.Header.Get(HeaderClientId)
	logger := s.logger.With("client-id", clientID)
	logger.Info("incoming connection from " + r.RemoteAddr)

	if s.requireClientId && clientID == "" {
		http.Error(w, fmt.Sprintf("header %s is required", HeaderClientId), http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("upgrade failed", slog.String("error", err.Error()))
		return
	}
	handleConn(s.parent, s.pool, clientID, conn, s.handler, s.codec, logger)
}

func (s *Serve) HandleProxy(w http.ResponseWriter, r *http.Request) {
	clientID := r.Header.Get(HeaderClientId)
	conn := s.GetConnByID(clientID)
	if conn == nil {
		msg := fmt.Sprintf("connection for clientID='%s' not found", clientID)
		s.logger.Error(msg)
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	err := s.handler.ProxyRequest(conn, w, r)
	if err != nil {
		s.logger.Error("proxy request failed", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
}

func (s *Serve) HandleProxyWithRetry(w http.ResponseWriter, r *http.Request) {
	clientID := r.Header.Get(HeaderClientId)
	conns := s.GetConnsByID(clientID)
	if len(conns) == 0 {
		msg := fmt.Sprintf("connection for clientID='%s' not found", clientID)
		s.logger.Error(msg)
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	var err error
	for _, conn := range conns {
		err = s.handler.ProxyRequest(conn, w, r)
		if err == nil {
			return
		}
		if errors.Is(err, ErrConnectionClosed) {
			continue
		}
	}
	if err != nil {
		s.logger.Error("proxy request failed", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
}
