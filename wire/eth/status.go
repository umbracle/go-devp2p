package eth

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/umbracle/ethgo"
	"github.com/umbracle/fastrlp"
)

type ForkID struct {
	Hash []byte // CRC32 checksum of the genesis block and passed fork block numbers
	Next uint64 // Block number of the next upcoming fork, or 0 if no forks are known
}

func (f *ForkID) Equal(ff *ForkID) bool {
	return f.Next != ff.Next || bytes.Equal(f.Hash, ff.Hash)
}

func (f *ForkID) UnmarshalRLPWith(v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	if len(elems) != 2 {
		return fmt.Errorf("bad length, expected 2 items but found %d", len(elems))
	}

	if f.Hash, err = elems[0].GetBytes(f.Hash, 4); err != nil {
		return err
	}
	if f.Next, err = elems[1].GetUint64(); err != nil {
		return err
	}
	return nil
}

func (f *ForkID) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	v := ar.NewArray()
	v.Set(ar.NewCopyBytes(f.Hash))
	v.Set(ar.NewUint(f.Next))
	return v
}

type Status struct {
	ProtocolVersion uint64
	NetworkID       uint64
	TD              *big.Int
	Head            [32]byte
	Genesis         [32]byte
	ForkID          ForkID
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

type TransactionsMsgPacket []*ethgo.Transaction

func (t *TransactionsMsgPacket) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	panic("TODO")
}

func (t *TransactionsMsgPacket) UnmarshalRLPWith(v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}

	for _, elem := range elems {
		txn := &ethgo.Transaction{}

		if elem.Type() == fastrlp.TypeBytes {
			// new types
			buf, _ := elem.GetBytes(nil)
			fmt.Println(buf[0])

			switch typ := buf[0]; typ {
			case 1:
				txn.Type = ethgo.TransactionAccessList
			case 2:
				txn.Type = ethgo.TransactionDynamicFee
			default:
				return fmt.Errorf("type byte %d not found", typ)
			}

			pp := fastrlp.Parser{}
			subVal, err := pp.Parse(buf[1:])
			if err != nil {
				panic(err)
			}
			if err := txn.UnmarshalRLPWith(subVal); err != nil {
				panic(err)
			}

		} else {
			// legacy
			if err := txn.UnmarshalRLPWith(elem); err != nil {
				panic(err)
			}
		}

		*t = append(*t, txn)
	}
	return nil
}

func (t *TransactionsMsgPacket) UnmarshalRLP(buf []byte) error {
	p := fastrlp.Parser{}

	v, err := p.Parse(buf)
	if err != nil {
		return err
	}

	if err := t.UnmarshalRLPWith(v); err != nil {
		return err
	}
	return nil
}
