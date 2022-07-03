package devp2p

import (
	"fmt"

	"github.com/umbracle/go-devp2p/enode"
)

type Status int

const (
	PeerActive Status = iota
	PeerPending
	PeerDisconnected
	PeerBusy
)

func (s Status) String() string {
	switch s {
	case PeerActive:
		return "active"

	case PeerPending:
		return "pending"

	case PeerDisconnected:
		return "disconnected"

	default:
		panic(fmt.Sprintf("Status %d not found", s))
	}
}

// Peer is each of the connected peers
type Peer struct {
	Enode     *enode.Enode
	Info      Info
	ID        string
	prettyID  string
	Status    Status
	conn      Session
	protocols []*Instance
}

func newPeer(conn Session) *Peer {
	info := conn.GetInfo()
	id := info.Enode.ID.String()

	peer := &Peer{
		Enode:     info.Enode,
		Info:      info,
		ID:        id,
		prettyID:  id[:8],
		conn:      conn,
		protocols: []*Instance{},
	}

	return peer
}

// Session returns the session of the peer
func (p *Peer) Session() Session {
	return p.conn
}

// GetProtocols returns all the protocols of the peer
func (p *Peer) GetProtocols() []*Instance {
	return p.protocols
}

// GetProtocol returns the protocol by name
func (p *Peer) GetProtocol(name string) (*Instance, bool) {
	for _, i := range p.protocols {
		if i.Protocol.Spec.Name == name {
			return i, true
		}
	}
	return nil, false
}

// IsClosed checks if the connection is closed
func (p *Peer) IsClosed() bool {
	return p.conn.IsClosed()
}

// PrettyID returns a pretty version of the id
func (p *Peer) PrettyID() string {
	return p.prettyID
}

// Close closes the peer connection
func (p *Peer) Close() error {
	return p.conn.Close()
}
