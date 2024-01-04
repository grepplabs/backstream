package ws

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/grepplabs/backstream/internal/message"
)

type Client struct {
	pool   *Pool
	parent context.Context
	urlStr string

	runOnce   func()
	connectMu sync.Mutex

	handler EventHandler

	codec Codec[*message.Message]

	logger        *slog.Logger
	clientID      string
	tlsConfigFunc func() *tls.Config
}

type ClientOption func(*Client)

func WithClientID(clientID string) ClientOption {
	return func(c *Client) {
		c.clientID = clientID
	}
}

func WithClientLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

func WithClientTLSConfigFunc(tlsConfigFunc func() *tls.Config) ClientOption {
	return func(c *Client) {
		c.tlsConfigFunc = tlsConfigFunc
	}
}

func NewClient(parent context.Context, urlStr string, handler EventHandler, codec Codec[*message.Message], opts ...ClientOption) *Client {
	client := &Client{
		pool:          NewPool(),
		parent:        parent,
		urlStr:        urlStr,
		handler:       handler,
		codec:         codec,
		logger:        slog.Default(),
		clientID:      "",
		tlsConfigFunc: nil,
	}
	for _, opt := range opts {
		opt(client)
	}
	client.runOnce = func() {
		go func() {
			client.keepConnected()
		}()
	}
	return client
}

func (c *Client) Start() {
	c.runOnce()
}

func (c *Client) GetConn() (*Conn, error) {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	client := c.pool.GetConn()
	if client != nil {
		return client, nil
	}
	return c.connect()
}

func (c *Client) keepConnected() {
	c.logger.Info("keep connected " + c.urlStr)
	_, err := c.GetConn()
	if err != nil {
		slog.Error("dial:" + err.Error())
	}
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-c.parent.Done():
			c.logger.Debug("context closed")
			return
		case <-ticker.C:
			// c.logger.Debug(fmt.Sprintf("client count (%d)", c.pool.Size()))
			_, err := c.GetConn()
			if err != nil {
				c.logger.Error("dial:" + err.Error())
			}
		}
	}
}

func (c *Client) connect() (*Conn, error) {
	c.logger.Info("connecting ws to " + c.urlStr)

	dialer := *websocket.DefaultDialer
	if c.tlsConfigFunc != nil {
		dialer.TLSClientConfig = c.tlsConfigFunc()
	}
	requestHeader := make(http.Header)
	requestHeader.Add(HeaderClientId, c.clientID)

	conn, resp, err := dialer.DialContext(c.parent, c.urlStr, requestHeader)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			b, err2 := io.ReadAll(resp.Body)
			if err2 != nil {
				return nil, fmt.Errorf("%w: status: %d, read body failed: %w", err, resp.StatusCode, err2)
			} else {
				return nil, fmt.Errorf("%w: status: %d, body: %s", err, resp.StatusCode, string(b))
			}
		} else {
			return nil, err
		}
	}
	return handleConn(c.parent, c.pool, c.clientID, conn, c.handler, c.codec, c.logger), nil
}
