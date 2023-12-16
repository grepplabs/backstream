package handler

import (
	"github.com/grepplabs/backstream/internal/message"
	"github.com/grepplabs/backstream/ws"
)

type HttpCodec interface {
	RequestCodec() ws.Codec[*message.EventHTTPRequest]
	ResponseCodec() ws.Codec[*message.EventHTTPResponse]
	MessageCodec() ws.Codec[*message.Message]
}

type httpJsonCodec struct {
	requestCodec  ws.Codec[*message.EventHTTPRequest]
	responseCodec ws.Codec[*message.EventHTTPResponse]
	messageCodec  ws.Codec[*message.Message]
}

func NewHttpJsonCodec() HttpCodec {
	return &httpJsonCodec{
		requestCodec:  ws.NewJsonCodec[*message.EventHTTPRequest](),
		responseCodec: ws.NewJsonCodec[*message.EventHTTPResponse](),
		messageCodec:  ws.NewJsonCodec[*message.Message](),
	}
}

func (c httpJsonCodec) RequestCodec() ws.Codec[*message.EventHTTPRequest] {
	return c.requestCodec
}

func (c httpJsonCodec) ResponseCodec() ws.Codec[*message.EventHTTPResponse] {
	return c.responseCodec
}

func (c httpJsonCodec) MessageCodec() ws.Codec[*message.Message] {
	return c.messageCodec
}

type httpProtoCodec struct {
	requestCodec  ws.Codec[*message.EventHTTPRequest]
	responseCodec ws.Codec[*message.EventHTTPResponse]
	messageCodec  ws.Codec[*message.Message]
}

func NewHttpProtoCodec() HttpCodec {
	return &httpProtoCodec{
		requestCodec:  ws.NewProtoCodec[*message.EventHTTPRequest](),
		responseCodec: ws.NewProtoCodec[*message.EventHTTPResponse](),
		messageCodec:  ws.NewProtoCodec[*message.Message](),
	}
}

func (c httpProtoCodec) RequestCodec() ws.Codec[*message.EventHTTPRequest] {
	return c.requestCodec
}

func (c httpProtoCodec) ResponseCodec() ws.Codec[*message.EventHTTPResponse] {
	return c.responseCodec
}

func (c httpProtoCodec) MessageCodec() ws.Codec[*message.Message] {
	return c.messageCodec
}
