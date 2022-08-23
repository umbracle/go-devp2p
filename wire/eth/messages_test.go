package eth

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/go-devp2p/forkid"
)

type codec interface {
	Marshaler
	Unmarshaler
}

func TestRLPEncoding(t *testing.T) {
	var cases = []codec{
		&HashList{
			[32]byte{0x1},
		},
		&Status{
			TD: big.NewInt(10),
			ForkID: forkid.ID{
				Hash: []byte{0x1, 0x2, 0x3, 0x4},
			},
		},
		&BlockHeadersPacket{},
		&BlockHeadersPacket{
			Hash: &([32]byte{0x1}),
		},
	}
	for _, obj := range cases {
		buf := MarshalRLP(obj)

		v := reflect.New(reflect.TypeOf(obj).Elem()).Interface()
		obj2 := v.(Unmarshaler)

		err := UnmarshalRLP(buf, obj2)
		require.NoError(t, err)
		require.Equal(t, obj, obj2)
	}
}
