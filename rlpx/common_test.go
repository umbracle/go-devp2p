package rlpx

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalUnmarshalInfo(t *testing.T) {

	info := &Info{
		// Version: 1,
		Name: "mock",
		//ListenPort: 30303,
		Caps: Capabilities{&Cap{"eth", 1}, &Cap{"par", 2}},
		// ID:         enode.PubkeyToEnode(&prv.PublicKey),
	}

	dst := info.MarshalRLP(nil)

	info2 := &Info{}
	if err := info2.UnmarshalRLP(dst); err != nil {
		t.Fatal(err)
	}

	fmt.Println(info)
	fmt.Println(info2)
	fmt.Println(reflect.DeepEqual(info, info2))
}

func TestDisconnectReason(t *testing.T) {
	reason := DiscQuitting

	for _, m := range [][]byte{
		{0xc1, byte(reason)},
		{byte(reason)},
	} {
		found, err := decodeDiscMsg(m)
		assert.NoError(t, err)
		assert.Equal(t, reason, found)
	}
}
