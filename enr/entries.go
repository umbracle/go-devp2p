package enr

import (
	"fmt"
	"net"

	"github.com/umbracle/fastrlp"
)

type Uint16 uint16

func (u Uint16) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	return ar.NewUint(uint64(u))
}

func (u *Uint16) UnmarshalRLPWith(v *fastrlp.Value) error {
	uu, err := v.GetUint64()
	if err != nil {
		return err
	}
	*u = Uint16(uu)
	return nil
}

type String string

func (s String) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	return ar.NewCopyBytes([]byte(s))
}

func (s *String) UnmarshalRLPWith(v *fastrlp.Value) error {
	buf, err := v.GetBytes(nil)
	if err != nil {
		return err
	}
	*s = String(buf)
	return nil
}

type IPv6 net.IP

func (i IPv6) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	ip := net.IP(i).To16()
	if ip == nil {
		return nil
	}
	return ar.NewCopyBytes(ip)
}

func (i *IPv6) UnmarshalRLPWith(v *fastrlp.Value) (err error) {
	*i, err = v.GetBytes(*i)
	if len(*i) != 4 {
		return fmt.Errorf("4 bytes expected for ipv4: %v", *v)
	}
	return err
}

type IPv4 net.IP

func (i IPv4) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	ip := net.IP(i).To4()
	if ip == nil {
		return nil
	}
	return ar.NewCopyBytes(ip)
}

func (i *IPv4) UnmarshalRLPWith(v *fastrlp.Value) (err error) {
	*i, err = v.GetBytes(*i)
	if len(*i) != 16 {
		return fmt.Errorf("16 bytes expected for ipv6: %v", *v)
	}
	return err
}
