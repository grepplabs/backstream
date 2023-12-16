package ws

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/grepplabs/backstream/internal/message"
	"github.com/grepplabs/backstream/internal/util"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 10 * time.Second
	// Time allowed to read the next pong message from the peer. Must be greater than pingPeriod.
	pongWait = (pingPeriod * 2) + time.Second
	// maximum message size allowed from peer.
	maxMessageSize = 10 * 1024 * 1024
)

var ErrConnectionClosed = errors.New("connection closed")

type EventHandler interface {
	HandleRequest(ctx context.Context, event []byte) ([]byte, error)
	HandleNotify(ctx context.Context, event []byte) error
}

type Conn struct {
	pool *Pool
	// client ID
	clientID string
	// The websocket connection.
	conn *websocket.Conn
	// Buffered channel of outbound messages.
	sendCh chan []byte
	// Buffered channels of response messages.
	respMap *util.SyncedMap[string, chan []byte]
	// Send channel closer
	sendClose func()
	// Cancel function
	cancel context.CancelFunc
	// Message Handler
	handler EventHandler
	// codec
	codec Codec[*message.Message]
	// logger
	logger *slog.Logger
}

func (c *Conn) readLoop(ctx context.Context) {
	defer func() {
		c.pool.unregister(c)
		c.sendClose()
		_ = c.conn.Close()
		c.logger.Debug("Reader closed")
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.logger.Debug("Received pong")
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		msgType, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("read message failure", slog.String("error", err.Error()))
			} else {
				c.logger.Warn("read message unexpected close", slog.String("error", err.Error()))
			}
			break
		}
		if msgType == websocket.BinaryMessage {
			c.logger.Debug("Received message")
		} else {
			c.logger.Debug("Received message : " + string(msg))
		}
		go func() {
			// long-lasting handleReceived blocks pong response as conn.ReadMessage() is not invoked
			resp := c.handleReceived(ctx, msg)
			if resp != nil {
				if msgType == websocket.BinaryMessage {
					c.logger.Debug("Sending response")
				} else {
					c.logger.Debug("Sending response : " + string(resp))
				}
				closed, err := safeSend(ctx, c.sendCh, resp)
				if err != nil {
					c.logger.Warn("readLoop send failure", slog.String("error", err.Error()))
					_ = c.conn.Close()
					return
				}
				if closed {
					c.logger.Warn("Channel closed")
					_ = c.conn.Close()
					return
				}
			}
		}()
	}
}

func (c *Conn) writeLoop(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
		c.logger.Debug("Writer closed")
	}()
	for {
		select {
		case <-ctx.Done():
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case msg, ok := <-c.sendCh:
			if c.codec.IsBinary() {
				c.logger.Debug("Sending")
			} else {
				c.logger.Debug("Sending : " + string(msg))
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The send channel was closed .
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			messageType := websocket.BinaryMessage
			if !c.codec.IsBinary() {
				messageType = websocket.TextMessage
			}
			w, err := c.conn.NextWriter(messageType)
			if err != nil {
				return
			}
			_, err = io.Copy(w, bytes.NewReader(msg))
			if err != nil {
				return
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.logger.Debug("Sending ping")
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Conn) Close() {
	c.logger.Debug("closing connection")
	c.cancel()
}

func (c *Conn) handleReceived(ctx context.Context, input []byte) []byte {
	var msg message.Message
	err := c.codec.Decode(input, &msg)
	if err != nil {
		c.logger.Error(err.Error())
		return nil
	}
	switch msg.Type {
	case message.Message_NOTIFY:
		err := c.handler.HandleNotify(ctx, msg.Data)
		if err != nil {
			return nil
		}
	case message.Message_REQUEST:
		output, err := c.handler.HandleRequest(ctx, msg.Data)
		if err != nil {
			return nil
		}
		msg.Data = output
		msg.Type = message.Message_RESPONSE
		data, err := c.codec.Encode(&msg)
		if err != nil {
			return nil
		}
		return data
	case message.Message_RESPONSE:
		// if no handlerFunc found means, that client received timeout and removed it
		if respCh, ok := c.respMap.Get(msg.Id); ok {
			respCh <- msg.Data
		}
	}
	return nil
}

func (c *Conn) Send(ctx context.Context, input []byte) ([]byte, error) {
	msg := &message.Message{
		Id:   uuid.New().String(),
		Type: message.Message_REQUEST,
		Data: input,
	}
	data, err := c.codec.Encode(msg)
	if err != nil {
		return nil, err
	}
	respCh := make(chan []byte, 1)
	c.respMap.Set(msg.Id, respCh)
	defer func() {
		c.respMap.Delete(msg.Id)
	}()

	closed, err := safeSend(ctx, c.sendCh, data)
	if err != nil {
		return nil, err
	}
	if closed {
		return nil, ErrConnectionClosed
	}
	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Conn) Notify(ctx context.Context, input []byte) error {
	msg := &message.Message{
		Id:   uuid.New().String(),
		Type: message.Message_NOTIFY,
		Data: input,
	}
	data, err := c.codec.Encode(msg)
	if err != nil {
		return err
	}

	closed, err := safeSend(ctx, c.sendCh, data)
	if err != nil {
		return err
	}
	if closed {
		return ErrConnectionClosed
	}
	return nil
}

func handleConn(parent context.Context, pool *Pool, clientID string, conn *websocket.Conn, handler EventHandler, codec Codec[*message.Message], logger *slog.Logger) *Conn {
	ctx, cancel := context.WithCancel(parent)

	const inFlightCount = 1024
	sendCh := make(chan []byte, inFlightCount)

	client := &Conn{
		pool:     pool,
		clientID: clientID,
		conn:     conn,
		respMap:  util.NewSyncedMap[string, chan []byte](),
		sendCh:   sendCh,
		sendClose: sync.OnceFunc(func() {
			close(sendCh)
		}),
		cancel:  cancel,
		handler: handler,
		codec:   codec,
		logger:  logger,
	}
	pool.register(client)

	go client.writeLoop(ctx)
	go client.readLoop(ctx)

	return client
}

func safeSend[T any](ctx context.Context, ch chan T, value T) (closed bool, err error) {
	defer func() {
		if recover() != nil {
			closed = true
		}
	}()

	select {
	case ch <- value:
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
