package ws

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Codec[T proto.Message] interface {
	Encode(t T) ([]byte, error)
	Decode(data []byte, t T) error
	// IsBinary determines is message is text or binary data message.
	IsBinary() bool
}

type JsonCodec[T proto.Message] struct {
}

func NewJsonCodec[T proto.Message]() Codec[T] {
	return &JsonCodec[T]{}
}

func (c JsonCodec[T]) Encode(v T) ([]byte, error) {
	return protojson.Marshal(v)
}

func (c JsonCodec[T]) Decode(data []byte, t T) error {
	return protojson.Unmarshal(data, t)
}
func (c JsonCodec[T]) IsBinary() bool {
	return false
}

type ProtoCodec[T proto.Message] struct {
}

func NewProtoCodec[T proto.Message]() Codec[T] {
	return &ProtoCodec[T]{}
}

func (c ProtoCodec[T]) Encode(v T) ([]byte, error) {
	return proto.Marshal(v)
}

func (c ProtoCodec[T]) Decode(data []byte, t T) error {
	return proto.Unmarshal(data, t)
}

func (c ProtoCodec[T]) IsBinary() bool {
	return true
}
