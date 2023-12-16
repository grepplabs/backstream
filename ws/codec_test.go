package ws

import (
	"testing"

	"github.com/google/uuid"
	"github.com/grepplabs/backstream/internal/message"
	"github.com/stretchr/testify/require"
)

func TestCodec(t *testing.T) {
	tests := []struct {
		name   string
		msg    *message.Message
		codec  Codec[*message.Message]
		binary bool
	}{
		{
			name: "Json Notify",
			msg: &message.Message{
				Id:   uuid.New().String(),
				Type: message.Message_NOTIFY,
				Data: nil,
			},
			codec: NewJsonCodec[*message.Message](),
		},
		{
			name: "Proto Notify",
			msg: &message.Message{
				Id:   uuid.New().String(),
				Type: message.Message_NOTIFY,
				Data: nil,
			},
			codec:  NewProtoCodec[*message.Message](),
			binary: true,
		},
		{
			name: "Json Request",
			msg: &message.Message{
				Id:   uuid.New().String(),
				Type: message.Message_REQUEST,
				Data: []byte{42},
			},
			codec: NewJsonCodec[*message.Message](),
		},
		{
			name: "Proto Request",
			msg: &message.Message{
				Id:   uuid.New().String(),
				Type: message.Message_REQUEST,
				Data: []byte{42},
			},
			codec:  NewProtoCodec[*message.Message](),
			binary: true,
		},
		{
			name: "Json Response",
			msg: &message.Message{
				Id:   uuid.New().String(),
				Type: message.Message_RESPONSE,
				Data: []byte{42},
			},
			codec: NewJsonCodec[*message.Message](),
		},
		{
			name: "Proto Response",
			msg: &message.Message{
				Id:   uuid.New().String(),
				Type: message.Message_RESPONSE,
				Data: []byte{42},
			},
			codec:  NewProtoCodec[*message.Message](),
			binary: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.codec.Encode(tc.msg)
			require.NoError(t, err)
			var msg message.Message
			err = tc.codec.Decode(data, &msg)
			require.NoError(t, err)
			require.Equal(t, tc.msg.Id, msg.Id)
			require.Equal(t, tc.msg.Type, msg.Type)
			require.Equal(t, tc.msg.Data, msg.Data)
			require.Equal(t, tc.binary, tc.codec.IsBinary())
		})
	}
}
