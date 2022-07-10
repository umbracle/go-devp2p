package eth

import (
	"fmt"
	"math/big"

	"github.com/umbracle/fastrlp"
	"github.com/umbracle/go-devp2p/forkid"
)

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

func (s *Status) UnmarshalRLP(buf []byte) error {
	p := fastrlp.Parser{}

	v, err := p.Parse(buf)
	if err != nil {
		return err
	}
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

func (s *Status) MarshalRLP() ([]byte, error) {
	ar := &fastrlp.Arena{}

	v := ar.NewArray()
	v.Set(ar.NewUint(s.ProtocolVersion))
	v.Set(ar.NewUint(s.NetworkID))
	v.Set(ar.NewBigInt(s.TD))
	v.Set(ar.NewBytes(s.Head[:]))
	v.Set(ar.NewBytes(s.Genesis[:]))
	v.Set(s.ForkID.MarshalRLPWith(ar))

	dst := v.MarshalTo(nil)
	return dst, nil
}
