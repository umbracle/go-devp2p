package eth

import (
	"fmt"
	"math/big"

	"github.com/umbracle/fastrlp"
	"github.com/umbracle/go-devp2p/forkid"
)

// Marshaler is the interface implemented by types that can marshal themselves into valid RLP messages.
type Marshaler interface {
	MarshalRLPWith(a *fastrlp.Arena) *fastrlp.Value
}

// Unmarshaler is the interface implemented by types that can unmarshal a RLP description of themselves
type Unmarshaler interface {
	UnmarshalRLPWith(v *fastrlp.Value) error
}

// MarshalRLP marshals an RLP object
func MarshalRLP(m Marshaler) []byte {
	ar := &fastrlp.Arena{}
	v := m.MarshalRLPWith(ar)
	return v.MarshalTo(nil)
}

// UnmarshalRLP unmarshals an RLP object
func UnmarshalRLP(buf []byte, m Unmarshaler) error {
	p := &fastrlp.Parser{}
	v, err := p.Parse(buf)
	if err != nil {
		return err
	}
	if err := m.UnmarshalRLPWith(v); err != nil {
		return err
	}
	return nil
}

type Request struct {
	RequestId uint64
	Body      Marshaler
}

func (e *Request) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	v := ar.NewArray()
	v.Set(ar.NewUint(e.RequestId))
	v.Set(e.Body.MarshalRLPWith(ar))
	return v
}

type Response struct {
	RequestId uint64
	Body      Unmarshaler
	BodyRaw   *fastrlp.Value
}

func (e *Response) UnmarshalRLPWith(v *fastrlp.Value) error {
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

func (n *newBlockHashesPacket) UnmarshalRLPWith(v *fastrlp.Value) error {
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

type Status struct {
	ProtocolVersion uint64
	NetworkID       uint64
	TD              *big.Int
	Head            [32]byte
	Genesis         [32]byte
	ForkID          forkid.ID
}

func (s *Status) Equal(ss *Status) error {
	if s.ProtocolVersion != ss.ProtocolVersion {
		return fmt.Errorf("incorrect protocol version")
	}
	if s.NetworkID != ss.NetworkID {
		return fmt.Errorf("incorrect network")
	}
	if s.Genesis != ss.Genesis {
		return fmt.Errorf("incorrect genesis")
	}
	if !s.ForkID.Equal(&ss.ForkID) {
		return fmt.Errorf("incorrect fork id")
	}
	return nil
}

func (s *Status) UnmarshalRLPWith(v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	if len(elems) != 6 {
		return fmt.Errorf("bad length, expected 5 items but found %d", len(elems))
	}

	if s.ProtocolVersion, err = elems[0].GetUint64(); err != nil {
		return err
	}
	if s.NetworkID, err = elems[1].GetUint64(); err != nil {
		return err
	}
	s.TD = new(big.Int)
	if err := elems[2].GetBigInt(s.TD); err != nil {
		return err
	}
	if err = elems[3].GetHash(s.Head[:]); err != nil {
		return err
	}
	if err = elems[4].GetHash(s.Genesis[:]); err != nil {
		return err
	}
	if err := s.ForkID.UnmarshalRLPWith(elems[5]); err != nil {
		return err
	}
	return nil
}

func (s *Status) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	v := ar.NewArray()
	v.Set(ar.NewUint(s.ProtocolVersion))
	v.Set(ar.NewUint(s.NetworkID))
	v.Set(ar.NewBigInt(s.TD))
	v.Set(ar.NewBytes(s.Head[:]))
	v.Set(ar.NewBytes(s.Genesis[:]))
	v.Set(s.ForkID.MarshalRLPWith(ar))
	return v
}
