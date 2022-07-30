package rlpx

import (
	"fmt"
	"strings"

	"github.com/umbracle/fastrlp"
	"github.com/umbracle/go-devp2p/enode"
)

const (
	BaseProtocolVersion    = 5
	BaseProtocolLength     = uint64(16)
	baseProtocolMaxMsgSize = 2 * 1024

	snappyProtocolVersion = 5
)

type DiscReason uint

const (
	DiscRequested DiscReason = iota
	DiscNetworkError
	DiscProtocolError
	DiscUselessPeer
	DiscTooManyPeers
	DiscAlreadyConnected
	DiscIncompatibleVersion
	DiscInvalidIdentity
	DiscQuitting
	DiscUnexpectedIdentity
	DiscSelf
	DiscReadTimeout
	DiscSubprotocolError = 0x10
	DiscUnknown          = 0x100
)

func (d DiscReason) String() string {
	switch d {
	case DiscRequested:
		return "disconnect requested"
	case DiscNetworkError:
		return "network error"
	case DiscProtocolError:
		return "breach of protocol"
	case DiscUselessPeer:
		return "useless peer"
	case DiscTooManyPeers:
		return "too many peers"
	case DiscAlreadyConnected:
		return "already connected"
	case DiscIncompatibleVersion:
		return "incompatible p2p protocol version"
	case DiscInvalidIdentity:
		return "invalid node identity"
	case DiscQuitting:
		return "client quitting"
	case DiscUnexpectedIdentity:
		return "unexpected identity"
	case DiscSelf:
		return "connected to self"
	case DiscReadTimeout:
		return "read timeout"
	case DiscSubprotocolError:
		return "subprotocol error"
	default:
		return fmt.Sprintf("unknown disconnect reason: %d", d)
	}
}

func (d DiscReason) Error() string {
	return d.String()
}

func decodeDiscMsg(buf []byte) (DiscReason, error) {
	p := &fastrlp.Parser{}

	v, err := p.Parse(buf)
	if err != nil {
		return DiscUnknown, err
	}

	var elem *fastrlp.Value

	elems, err := v.GetElems()
	if err == nil {
		// the disconnect reason is an array
		if len(elems) != 1 {
			return DiscUnknown, fmt.Errorf("expected one item")
		}
		elem = elems[0]
	} else {
		// the disconnect reason is in plain bytes
		elem = v
	}

	num, err := elem.GetUint64()
	if err != nil {
		return DiscUnknown, fmt.Errorf("failed to decode disc reason uint64: %v", err)
	}

	reason := DiscReason(num)
	return reason, nil
}

const (
	// devp2p message codes
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
)

// Cap is the peer capability.
type Cap struct {
	Name    string
	Version uint64
}

func (c *Cap) unmarshalRLP(v *fastrlp.Value) error {
	fields, err := v.GetElems()
	if err != nil {
		return err
	}
	if len(fields) != 2 {
		return fmt.Errorf("bad parsing")
	}
	if c.Name, err = fields[0].GetString(); err != nil {
		return err
	}
	if c.Version, err = fields[1].GetUint64(); err != nil {
		return err
	}
	return nil
}

func (c *Cap) less(cc *Cap) bool {
	if cmp := strings.Compare(c.Name, cc.Name); cmp != 0 {
		return cmp == -1
	}
	return c.Version < cc.Version
}

// Capabilities are all the capabilities of the other peer
type Capabilities []*Cap

func (c Capabilities) Len() int {
	return len(c)
}

func (c Capabilities) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Capabilities) Less(i, j int) bool {
	return c[i].less(c[j])
}

// Info is the info of the node
type Info struct {
	Version    uint64
	Name       string
	Caps       Capabilities
	ListenPort uint64
	ID         enode.ID
}

const nodeIDBytes = 512 / 8

var infoArenaPool fastrlp.ArenaPool

func (i *Info) MarshalRLP(dst []byte) []byte {
	a := infoArenaPool.Get()

	v := a.NewArray()
	v.Set(a.NewUint(i.Version))
	v.Set(a.NewString(i.Name))
	if len(i.Caps) == 0 {
		v.Set(a.NewNullArray())
	} else {
		caps := a.NewArray()
		for _, cap := range i.Caps {
			elem := a.NewArray()
			elem.Set(a.NewString(cap.Name))
			elem.Set(a.NewUint(cap.Version))
			caps.Set(elem)
		}
		v.Set(caps)
	}
	v.Set(a.NewUint(i.ListenPort))
	v.Set(a.NewBytes(i.ID[:]))

	dst = v.MarshalTo(dst)
	infoArenaPool.Put(a)
	return dst
}

var infoParserPool fastrlp.ParserPool

func (i *Info) UnmarshalRLP(b []byte) error {
	p := infoParserPool.Get()
	defer infoParserPool.Put(p)

	v, err := p.Parse(b)
	if err != nil {
		return err
	}
	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	if len(elems) < 5 {
		// there might be additional fields for forward compatibility
		return fmt.Errorf("bad length cc")
	}

	if i.Version, err = elems[0].GetUint64(); err != nil {
		return err
	}
	if i.Name, err = elems[1].GetString(); err != nil {
		return err
	}

	// array of capabilities
	caps, err := elems[2].GetElems()
	if err != nil {
		return err
	}
	for _, elem := range caps {
		cap := &Cap{}
		if err := cap.unmarshalRLP(elem); err != nil {
			return err
		}
		i.Caps = append(i.Caps, cap)
	}

	// listen port
	if i.ListenPort, err = elems[3].GetUint64(); err != nil {
		return err
	}

	// enode
	if elems[4].Type() != fastrlp.TypeBytes {
		return fmt.Errorf("bad")
	}
	if len(elems[4].Raw()) != nodeIDBytes {
		return fmt.Errorf("bad.2")
	}
	copy(i.ID[:], elems[4].Raw())

	return nil
}
