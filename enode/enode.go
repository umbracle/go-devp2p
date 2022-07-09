package enode

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/umbracle/fastrlp"
	"github.com/umbracle/go-devp2p/crypto"
	"github.com/umbracle/go-devp2p/enr"
)

const nodeIDBytes = 512 / 8

// ID is the unique identifier of each node.
type ID [nodeIDBytes]byte

func (i ID) String() string {
	return hex.EncodeToString(i[:])
}

func (i ID) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	return ar.NewCopyBytes(i[:])
}

func (id *ID) UnmarshalRLPWith(v *fastrlp.Value) (err error) {
	if _, err := v.GetBytes(id[:], nodeIDBytes); err != nil {
		return err
	}
	return nil
}

// Enode is the URL scheme description of an ethereum node.
type Enode struct {
	ID ID
	r  *enr.Record
}

func New(ip net.IP, tcpPort, udpPort uint16, id ID) *Enode {
	r := &enr.Record{}

	tcpEntry := enr.Uint16(tcpPort)
	r.AddEntry("tcp", &tcpEntry)

	udpEntry := enr.Uint16(udpPort)
	r.AddEntry("udp", &udpEntry)

	ipEntry := enr.IPv4(ip)
	r.AddEntry("ip", &ipEntry)

	node := &Enode{
		ID: id,
		r:  r,
	}
	return node
}

func NewFromEnr(record *enr.Record) (*Enode, error) {
	return nil, nil
}

// ParseURL parses an node address either in enode or enr format
func NewFromURL(rawurl string) (*Enode, error) {
	if strings.HasPrefix(rawurl, "enr:") {
		record, err := enr.Unmarshal(rawurl)
		if err != nil {
			return nil, err
		}
		return NewFromEnr(record)
	}

	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "enode" {
		return nil, fmt.Errorf("invalid URL scheme, expected 'enode'")
	}

	var id ID
	h, err := hex.DecodeString(u.User.String())
	if err != nil {
		return nil, fmt.Errorf("failed to decode id: %v", err)
	}
	if len(h) != nodeIDBytes {
		return nil, fmt.Errorf("id not found")
	}
	copy(id[:], h)

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host: %v", err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address '%s'", host)
	}

	tcpPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid tcp port '%s': %v", port, err)
	}

	udpPort := tcpPort
	if discPort := u.Query().Get("discport"); discPort != "" {
		udpPort, err = strconv.ParseUint(discPort, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid udp port '%s': %v", discPort, err)
		}
	}

	node := New(ip, uint16(tcpPort), uint16(udpPort), id)
	return node, nil
}

func (n *Enode) String() string {
	url := fmt.Sprintf("enode://%s@%s", n.ID.String(), (&net.TCPAddr{IP: n.IP(), Port: int(n.TCP())}).String())
	if n.TCP() != n.UDP() {
		url += "?discport=" + strconv.Itoa(int(n.UDP()))
	}
	return url
}

func (n *Enode) IP() net.IP {
	var ip4 enr.IPv4
	if err := n.r.Load("ip", &ip4); err == nil {
		return net.IP(ip4)
	}
	var ip6 enr.IPv6
	if err := n.r.Load("ip", &ip6); err == nil {
		return net.IP(ip6)
	}
	return nil
}

func (n *Enode) UDP() uint16 {
	var udpPort enr.Uint16
	n.r.Load("udp", &udpPort)
	return uint16(udpPort)
}

func (n *Enode) TCP() uint16 {
	var tcpPort enr.Uint16
	n.r.Load("tcp", &tcpPort)
	return uint16(tcpPort)
}

// PublicKey returns the public key of the enode
func (n *Enode) PublicKey() (*ecdsa.PublicKey, error) {
	return NodeIDToPubKey(n.ID[:])
}

// TCPAddr returns the TCP address
func (n *Enode) TCPAddr() net.TCPAddr {
	return net.TCPAddr{IP: n.IP(), Port: int(n.TCP())}
}

// NodeIDToPubKey returns the public key of the enode ID
func NodeIDToPubKey(buf []byte) (*ecdsa.PublicKey, error) {
	if len(buf) != nodeIDBytes {
		return nil, fmt.Errorf("not enough length: expected %d but found %d", nodeIDBytes, len(buf))
	}
	p := &ecdsa.PublicKey{Curve: crypto.S256, X: new(big.Int), Y: new(big.Int)}
	half := len(buf) / 2
	p.X.SetBytes(buf[:half])
	p.Y.SetBytes(buf[half:])
	if !p.Curve.IsOnCurve(p.X, p.Y) {
		return nil, errors.New("id is invalid secp256k1 curve point")
	}
	return p, nil
}

// PubkeyToEnode converts a public key to an enode
func PubkeyToEnode(pub *ecdsa.PublicKey) ID {
	var id ID
	pbytes := crypto.MarshallPublicKey(pub)
	copy(id[:], pbytes[1:])
	return id
}
