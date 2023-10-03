package protobuf_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/p2p/host/autonat/pb"
	"github.com/primevprotocol/mev-commit/pkg/p2p/protobuf"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
	"google.golang.org/protobuf/proto"
)

func TestProtobufEncodingDecoding(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		out, in := p2ptest.NewDuplexStream()

		test := &pb.Message{
			Type: pb.Message_DIAL.Enum(),
			Dial: &pb.Message_Dial{
				Peer: &pb.Message_PeerInfo{
					Id: []byte("16Uiu2HAmK8EQ9axsSaE9hqjdHX7Hq5Jbeo2tmuNcLHwyQLWKjSYw"),
					Addrs: [][]byte{
						[]byte("0x9Bbc6Bef724d483C8f834C03fC2D3FE115D47ABF"),
						[]byte("0x903e2Abdc0fF09aBCB4C23CD8Ef1e267dfD32c2C"),
						[]byte("0xdCFA8524A3A266A388A4884cB6448463ae19D025"),
					},
				},
			},
		}

		reader := protobuf.NewReaderWriter(in)
		writer := protobuf.NewReaderWriter(out)

		if err := writer.WriteMsg(context.Background(), test); err != nil {
			t.Fatal(err)
		}

		var res pb.Message
		err := reader.ReadMsg(context.Background(), &res)
		if err != nil {
			t.Fatal(err)
		}

		testBytes, err := proto.Marshal(test)
		if err != nil {
			t.Fatal(err)
		}

		resBytes, err := proto.Marshal(&res)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(testBytes, resBytes) {
			t.Fatalf("expected %v, got %v", testBytes, resBytes)
		}
	})
}
