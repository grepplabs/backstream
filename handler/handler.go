package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/grepplabs/backstream/internal/message"
	"github.com/grepplabs/backstream/ws"
)

const (
	HeaderRequestTimeout  = "x-backstream-request-timeout"
	DefaultRequestTimeout = 3 * time.Second
)

type proxyHandler struct {
	codec                 HttpCodec
	defaultRequestTimeout time.Duration
}

type ProxyHandlerOption func(*proxyHandler)

func WithProxyDefaultRequestTimeout(timeout time.Duration) ProxyHandlerOption {
	return func(c *proxyHandler) {
		c.defaultRequestTimeout = timeout
	}
}

func NewProxyHandler(codec HttpCodec, opts ...ProxyHandlerOption) ws.ProxyHandler {
	h := &proxyHandler{
		codec:                 codec,
		defaultRequestTimeout: DefaultRequestTimeout,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *proxyHandler) HandleRequest(_ context.Context, _ []byte) ([]byte, error) {
	return nil, errors.New("proxy request is not implemented")
}

func (h *proxyHandler) HandleNotify(_ context.Context, _ []byte) error {
	return errors.New("proxy notification is not implemented")
}

func (h *proxyHandler) ProxyRequest(conn *ws.Conn, w http.ResponseWriter, r *http.Request) error {
	return ProxyHttpRequest(conn, w, r, h.codec, h.defaultRequestTimeout)
}

func GetRequestTimeout(r *http.Request, defaultRequestTimeout time.Duration) (time.Duration, error) {
	var requestTimeout time.Duration
	if rt := r.Header.Get(HeaderRequestTimeout); rt != "" {
		var err error
		if requestTimeout, err = time.ParseDuration(rt); err != nil {
			return 0, err
		}
	} else {
		requestTimeout = defaultRequestTimeout
	}
	return requestTimeout, nil
}

func ProxyHttpRequest(conn *ws.Conn, w http.ResponseWriter, r *http.Request, codec HttpCodec, defaultRequestTimeout time.Duration) error {
	inputEvent, err := fromHttpRequest(r)
	if err != nil {
		return err
	}
	input, err := codec.RequestCodec().Encode(inputEvent)
	if err != nil {
		return err
	}

	ctx := r.Context()
	requestTimeout, err := GetRequestTimeout(r, defaultRequestTimeout)
	if err != nil {
		return err
	}
	if requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, requestTimeout)
		defer cancel()
	}

	output, err := conn.Send(ctx, input)
	if err != nil {
		return err
	}
	var outputEvent message.EventHTTPResponse
	err = codec.ResponseCodec().Decode(output, &outputEvent)
	if err != nil {
		return err
	}
	return writeHttpResponse(w, &outputEvent)
}

type recoveryHandler struct {
	target ws.EventHandler
	logger *slog.Logger
}

func NewRecoveryHandler(target ws.EventHandler, logger *slog.Logger) ws.EventHandler {
	return &recoveryHandler{
		target: target,
		logger: logger,
	}
}

func (h *recoveryHandler) HandleRequest(ctx context.Context, event []byte) ([]byte, error) {
	defer func() {
		if err := recover(); err != nil {
			h.logger.Error("[Recovery] panic recovered", slog.String("error", fmt.Sprintf("%v", err)))
		}
	}()
	return h.target.HandleRequest(ctx, event)
}

func (h *recoveryHandler) HandleNotify(ctx context.Context, event []byte) error {
	defer func() {
		if err := recover(); err != nil {
			h.logger.Error("[Recovery] panic recovered", slog.String("error", fmt.Sprintf("%v", err)))
		}
	}()
	return h.target.HandleNotify(ctx, event)
}

type HTTPHandler struct {
	handlerFunc           http.HandlerFunc
	codec                 HttpCodec
	defaultRequestTimeout time.Duration
}

type HTTPHandlerOption func(*HTTPHandler)

func WithHTTPDefaultRequestTimeout(timeout time.Duration) HTTPHandlerOption {
	return func(c *HTTPHandler) {
		c.defaultRequestTimeout = timeout
	}
}

func NewHTTPHandler(handlerFunc http.HandlerFunc, codec HttpCodec, opts ...HTTPHandlerOption) *HTTPHandler {
	h := &HTTPHandler{
		handlerFunc:           handlerFunc,
		codec:                 codec,
		defaultRequestTimeout: 0,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *HTTPHandler) HandleRequest(ctx context.Context, event []byte) ([]byte, error) {
	return HttpRequestHandler(ctx, h.handlerFunc, event, h.codec, h.defaultRequestTimeout)
}

func (h *HTTPHandler) HandleNotify(ctx context.Context, event []byte) error {
	return HttpNotifyHandler(ctx, h.handlerFunc, event, h.codec, h.defaultRequestTimeout)
}

func HttpRequestHandler(ctx context.Context, handler http.HandlerFunc, event []byte, codec HttpCodec, defaultRequestTimeout time.Duration) ([]byte, error) {
	var inputEvent message.EventHTTPRequest
	err := codec.RequestCodec().Decode(event, &inputEvent)
	if err != nil {
		return nil, err
	}
	req, err := toHttpRequest(&inputEvent)
	if err != nil {
		return nil, err
	}

	requestTimeout, err := GetRequestTimeout(req, defaultRequestTimeout)
	if err != nil {
		return nil, err
	}
	if requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, requestTimeout)
		defer cancel()
	}
	req = req.WithContext(ctx)

	// process
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()

	outputEvent, err := fromHttpResponse(resp)
	if err != nil {
		return nil, err
	}
	output, err := codec.ResponseCodec().Encode(outputEvent)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func HttpNotifyHandler(ctx context.Context, handler http.HandlerFunc, event []byte, codec HttpCodec, defaultRequestTimeout time.Duration) error {
	var inputEvent message.EventHTTPRequest
	err := codec.RequestCodec().Decode(event, &inputEvent)
	if err != nil {
		return err
	}
	req, err := toHttpRequest(&inputEvent)
	if err != nil {
		return err
	}
	requestTimeout, err := GetRequestTimeout(req, defaultRequestTimeout)
	if err != nil {
		return err
	}
	if requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, requestTimeout)
		defer cancel()
	}
	req = req.WithContext(ctx)

	// process
	w := httptest.NewRecorder()
	handler(w, req)

	return nil
}
