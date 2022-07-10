package eth

import (
	"fmt"

	"github.com/umbracle/fastrlp"
)

type Eth66Body interface {
	MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value
	UnmarshalRLPWith(v *fastrlp.Value) error
}

type RlpRequest interface {
	MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value
}

type RlpResponse interface {
	UnmarshalRLPWith(v *fastrlp.Value) error
}

type Request struct {
	RequestId uint64
	Body      RlpRequest
}

func (e *Request) MarshalRLP() ([]byte, error) {
	ar := &fastrlp.Arena{}

	v := ar.NewArray()
	v.Set(ar.NewUint(e.RequestId))
	v.Set(e.Body.MarshalRLPWith(ar))

	dst := v.MarshalTo(nil)
	return dst, nil
}

type Response struct {
	RequestId uint64
	Body      RlpResponse
	BodyRaw   *fastrlp.Value
}

func (e *Response) UnmarshalRLP(buf []byte) error {
	p := &fastrlp.Parser{}
	v, err := p.Parse(buf)
	if err != nil {
		return err
	}

	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	if len(elems) != 2 {
		return fmt.Errorf("two items expected")
	}

	// decode the first Request Id
	if e.RequestId, err = elems[0].GetUint64(); err != nil {
		return err
	}
	// decode the body
	if e.Body != nil {
		if err := e.Body.UnmarshalRLPWith(elems[1]); err != nil {
			return err
		}
	} else {
		e.BodyRaw = elems[1]
	}
	return nil
}

type EmptyArray struct{}

func (e *EmptyArray) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	v := ar.NewNullArray()
	return v
}

func (e *EmptyArray) UnmarshalRLPWith(v *fastrlp.Value) error {
	panic("unimplemented")
}

type BlockHeadersPacket struct {
	// Origin, either Hash or Number
	Hash   *[32]byte
	Number uint64

	Amount  uint64
	Skip    uint64
	Reverse bool
}

func (g *BlockHeadersPacket) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	v := ar.NewArray()

	if g.Hash != nil {
		v.Set(ar.NewCopyBytes(g.Hash[:]))
	} else {
		v.Set(ar.NewUint(g.Number))
	}
	v.Set(ar.NewUint(g.Amount))
	v.Set(ar.NewUint(g.Skip))
	v.Set(ar.NewBool(g.Reverse))

	return v
}

func (g *BlockHeadersPacket) UnmarshalRLPWith(v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	if len(elems) != 4 {
		return fmt.Errorf("4 items expected but found %d", len(elems))
	}

	buf, err := elems[0].Bytes()
	if err != nil {
		return err
	}
	if len(buf) == 32 {
		// hash
		hash := [32]byte{}
		copy(hash[:], buf)
		g.Hash = &hash
	} else {
		// number
		if g.Number, err = elems[0].GetUint64(); err != nil {
			return err
		}
	}

	g.Amount, err = elems[1].GetUint64()
	if err != nil {
		return err
	}
	g.Skip, err = elems[2].GetUint64()
	if err != nil {
		return err
	}
	g.Reverse, err = elems[3].GetBool()
	if err != nil {
		return err
	}
	return nil
}

type HashList [][32]byte

func (h *HashList) UnmarshalRLP(buf []byte) error {
	p := fastrlp.Parser{}

	v, err := p.Parse(buf)
	if err != nil {
		return err
	}
	if err := h.UnmarshalRLPWith(v); err != nil {
		return err
	}
	return nil
}

func (h *HashList) UnmarshalRLPWith(v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}

	for _, elem := range elems {
		var hash [32]byte
		if err = elem.GetHash(hash[:]); err != nil {
			return err
		}
		*h = append(*h, hash)
	}
	return nil
}

func (h *HashList) MarshalRLP() ([]byte, error) {
	ar := &fastrlp.Arena{}
	v := h.MarshalRLPWith(ar)
	res := v.MarshalTo(nil)
	return res, nil
}

func (h *HashList) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	v := ar.NewArray()
	for _, elem := range *h {
		v.Set(ar.NewCopyBytes(elem[:]))
	}
	return v
}

type RlpList []*fastrlp.Value

func (h *RlpList) UnmarshalRLPWith(v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	*h = elems
	return nil
}

type newBlockHashesPacket []*NewBlockHash

type NewBlockHash struct {
	Hash   [32]byte
	Number uint64
}

func (n *NewBlockHash) UnmarshalRLPWith(v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	if len(elems) != 2 {
		return fmt.Errorf("wrong num of elements, expected 2 but found %d", len(elems))
	}
	if err = elems[0].GetHash(n.Hash[:]); err != nil {
		return err
	}
	if n.Number, err = elems[1].GetUint64(); err != nil {
		return err
	}
	return nil
}

func (n *newBlockHashesPacket) UnmarshalRLP(buf []byte) error {
	p := fastrlp.Parser{}
	v, err := p.Parse(buf)
	if err != nil {
		return err
	}
	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	for _, elem := range elems {
		packet := &NewBlockHash{}
		if err := packet.UnmarshalRLPWith(elem); err != nil {
			return err
		}
		*n = append(*n, packet)
	}
	return nil
}
