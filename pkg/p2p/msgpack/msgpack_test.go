package msgpack_test

import (
	"context"
	"testing"

	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
)

func TestMsgpackEncoding(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		out, in := p2ptest.NewDuplexStream()

		type testStruct struct {
			Str string
			Num int
		}

		test := testStruct{
			Str: "test",
			Num: 1,
		}

		r, _ := msgpack.NewReaderWriter[testStruct, testStruct](in)
		_, w := msgpack.NewReaderWriter[testStruct, testStruct](out)

		if err := w.WriteMsg(context.Background(), &test); err != nil {
			t.Fatal(err)
		}

		res, err := r.ReadMsg(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if res.Str != test.Str {
			t.Fatalf("expected %s, got %s", test.Str, res.Str)
		}

		if res.Num != test.Num {
			t.Fatalf("expected %d, got %d", test.Num, res.Num)
		}
	})
}
