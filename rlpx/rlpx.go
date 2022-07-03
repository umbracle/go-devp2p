package rlpx

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/umbracle/go-devp2p"
	"github.com/umbracle/go-devp2p/enode"
)

// Rlpx is the RLPx transport protocol
type Rlpx struct {
	logger *log.Logger

	priv     *ecdsa.PrivateKey
	backends []*devp2p.Protocol
	info     *devp2p.Info

	addr string
	port int

	listener   net.Listener
	sessionCh  chan *Session
	shutdownCh chan struct{}
}

// getProtocol returns a protocol
func (r *Rlpx) getProtocol(name string, version uint) *devp2p.Protocol {
	for _, p := range r.backends {
		proto := p.Spec
		// fmt.Println(proto.Name, proto.Version, name, version)
		if proto.Name == name && proto.Version == version {
			return p
		}
	}
	return nil
}

// Connect implements the connect interface
func (r *Rlpx) connect(rawConn net.Conn, enode enode.Enode) (devp2p.Session, error) {
	pub, err := enode.PublicKey()
	if err != nil {
		return nil, err
	}

	conn := Client(r, rawConn, r.priv, pub, networkInfoToLocalInfo(r.info))
	if err := conn.Handshake(); err != nil {
		rawConn.Close()
		return conn, err
	}
	if err := conn.negotiateProtocols(); err != nil {
		return nil, err
	}
	return conn, nil
}

func (r *Rlpx) accept(rawConn net.Conn) (*Session, error) {
	conn := Server(r, rawConn, r.priv, networkInfoToLocalInfo(r.info))
	if err := conn.Handshake(); err != nil {
		rawConn.Close()
		return nil, err
	}
	if err := conn.negotiateProtocols(); err != nil {
		return nil, err
	}
	return conn, nil
}

// Setup implements the transport interface
func (r *Rlpx) Setup(priv *ecdsa.PrivateKey, backends []*devp2p.Protocol, info *devp2p.Info, config map[string]interface{}) error {
	r.priv = priv
	r.backends = backends
	r.info = info

	r.shutdownCh = make(chan struct{})

	if r.logger == nil {
		r.logger = log.New(ioutil.Discard, "", 0)
	}

	// TODO, safe check
	r.addr = config["addr"].(string)
	r.port = config["port"].(int)

	addr := net.TCPAddr{IP: net.ParseIP(r.addr), Port: r.port}

	r.logger.Printf("[INFO] Listening: addr, %s", addr.String())

	var err error
	r.listener, err = net.Listen("tcp", addr.String())
	if err != nil {
		return err
	}

	r.sessionCh = make(chan *Session, 10)

	go func() {
		for {
			conn, err := r.listener.Accept()
			if err != nil {
				return
			}
			go func() {
				session, err := r.accept(conn)
				if err != nil {
					// log
				} else {
					select {
					case r.sessionCh <- session:
					default:
					}
				}
			}()
		}
	}()

	return nil
}

// Server returns a new Rlpx server side Session
func Server(rlpx *Rlpx, conn net.Conn, prv *ecdsa.PrivateKey, info *Info) *Session {
	return &Session{rlpx: rlpx, conn: conn, prv: prv, Info: info}
}

// Client returns a new Rlpx client side Session
func Client(rlpx *Rlpx, conn net.Conn, prv *ecdsa.PrivateKey, pub *ecdsa.PublicKey, info *Info) *Session {
	return &Session{rlpx: rlpx, conn: conn, prv: prv, pub: pub, Info: info, isClient: true}
}

// DialTimeout implements the transport interface
func (r *Rlpx) DialTimeout(address string, timeout time.Duration) (devp2p.Session, error) {
	addr, err := enode.ParseURL(address)
	if err != nil {
		return nil, err
	}

	tcpAddr := addr.TCPAddr()
	conn, err := net.DialTimeout("tcp", tcpAddr.String(), timeout)
	if err != nil {
		return nil, err
	}

	session, err := r.connect(conn, *addr)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// Accept accepts a new incomming connection
func (r *Rlpx) Accept() (devp2p.Session, error) {
	select {
	case s := <-r.sessionCh:
		return s, nil
	case <-r.shutdownCh:
		return nil, fmt.Errorf("session closed")
	}
}

func (r *Rlpx) Close() error {
	close(r.shutdownCh)
	return r.listener.Close()
}

// networkInfoToLocalInfo converts the network info message into rlpx.Info
func networkInfoToLocalInfo(info *devp2p.Info) *Info {
	rlpxInfo := &Info{
		Version:    BaseProtocolVersion,
		Name:       info.Client,
		ListenPort: uint64(info.Enode.TCP),
		ID:         info.Enode.ID,
	}
	for _, cap := range info.Capabilities {
		p := cap.Protocol.Spec
		rlpxInfo.Caps = append(rlpxInfo.Caps, &Cap{Name: p.Name, Version: uint64(p.Version)})
	}
	return rlpxInfo
}
