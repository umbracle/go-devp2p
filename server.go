package devp2p

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/umbracle/go-devp2p/discovery"
	"github.com/umbracle/go-devp2p/enode"
)

// Protocol is a wire protocol
type Protocol struct {
	Spec      ProtocolSpec
	HandlerFn func(conn Stream, peer *Peer) error
}

// ProtocolSpec is a specification of an etheruem protocol
type ProtocolSpec struct {
	Name    string
	Version uint
	Length  uint64
}

// Info is the information of a peer
type Info struct {
	Client       string
	Enode        *enode.Enode
	Capabilities Capabilities
	ListenPort   uint64
}

// Capability is a feature of the peer
type Capability struct {
	Protocol Protocol
}

// Capabilities is a list of capabilities of the peer
type Capabilities []*Capability

type Instance struct {
	Protocol *Protocol
}

const (
	defaultDialTimeout = 10 * time.Second
	defaultDialTasks   = 15
)

type EventType int

const (
	NodeJoin EventType = iota
	NodeLeave
	NodeHandshakeFail
)

func (t EventType) String() string {
	switch t {
	case NodeJoin:
		return "node join"
	case NodeLeave:
		return "node leave"
	case NodeHandshakeFail:
		return "node handshake failed"
	default:
		panic(fmt.Sprintf("unknown event type: %d", t))
	}
}

type MemberEvent struct {
	Type EventType
	Peer *Peer
}

// Server is the ethereum client
type Server struct {
	logger *log.Logger
	Name   string
	key    *ecdsa.PrivateKey

	peersLock sync.Mutex
	peers     map[string]*Peer

	info *Info

	config  *Config
	closeCh chan struct{}
	EventCh chan MemberEvent

	// set of pending nodes
	pendingNodes sync.Map

	addPeer chan string

	dispatcher *Dispatcher

	peerStore PeerStore
	transport Transport

	Discovery discovery.Discovery
	Enode     *enode.Enode
}

// NewServer creates a new node
func NewServer(key *ecdsa.PrivateKey, transport Transport, opts ...ConfigOption) (*Server, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	enode := enode.New(
		net.ParseIP(config.BindAddress),
		uint16(config.BindPort),
		uint16(config.BindPort),
		enode.PubkeyToEnode(&key.PublicKey),
	)

	s := &Server{
		Name:         config.Name,
		key:          key,
		peers:        map[string]*Peer{},
		peersLock:    sync.Mutex{},
		config:       config,
		logger:       config.Logger,
		closeCh:      make(chan struct{}),
		Enode:        enode,
		EventCh:      make(chan MemberEvent, 20),
		pendingNodes: sync.Map{},
		addPeer:      make(chan string, 20),
		dispatcher:   NewDispatcher(),
		peerStore:    &NoopPeerStore{},
		transport:    transport,
	}

	// setup discovery
	if err := s.setupDiscovery(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) setupDiscovery() error {
	// setup discovery factories
	discoveryConfig := &discovery.DiscoveryConfig{
		Key:       s.key,
		Enode:     s.Enode,
		Bootnodes: s.config.Bootnodes,
	}

	discovery, err := discovery.DiscV4(context.Background(), discoveryConfig)
	if err != nil {
		return err
	}
	s.Discovery = discovery
	return nil
}

// GetPeers returns a copy of list of peers
func (s *Server) GetPeers() []string {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	ids := []string{}
	for id := range s.peers {
		ids = append(ids, id)
	}
	return ids
}

func (s *Server) buildInfo() {
	info := &Info{
		Client: s.Name,
		Enode:  s.Enode,
	}

	for _, p := range s.config.Protocols {
		cap := &Capability{
			Protocol: *p,
		}

		info.Capabilities = append(info.Capabilities, cap)
	}
	s.info = info
}

// Schedule starts all the tasks once all the protocols have been loaded
func (s *Server) Start() error {
	// bootstrap peers
	storedPeers, err := s.peerStore.Load()
	if err != nil {
		return err
	}
	for _, peer := range storedPeers {
		s.Dial(peer)
	}

	// Create rlpx info
	s.buildInfo()

	config := map[string]interface{}{
		"addr": s.config.BindAddress,
		"port": s.config.BindPort,
	}

	if err := s.transport.Setup(s.key, s.config.Protocols, s.info, config); err != nil {
		return err
	}

	go func() {
		session, err := s.transport.Accept()
		if err == nil {
			if err := s.addSession(session); err != nil {
				// log
			}
		}
	}()

	// Start discovery process
	s.Discovery.Schedule()

	go s.dialRunner()
	return nil
}

// PeriodicDial is the periodic dial of busy peers
type PeriodicDial struct {
	enode string
}

// ID returns the id of the enode
func (p *PeriodicDial) ID() string {
	return p.enode
}

// -- DIALING --

func (s *Server) dialTask(id string, tasks chan string) {
	// s.logger.Printf("Dial task %s running", id)

	for {
		select {
		case task := <-tasks:
			s.logger.Printf("[TRACE]: DIAL: id %s task %s", id, task)

			err := s.connect(task)

			contains := s.dispatcher.Contains(task)
			busy := false
			if err != nil {
				s.logger.Printf("[ERROR]: dial: id, %s, err, %v", id, err)

				if err.Error() == "too many peers" {
					busy = true
				}
			}

			if busy {
				// the peer had too many peers, reschedule to dial it again if it is not already on the list
				if !contains {
					if err := s.dispatcher.Add(&PeriodicDial{task}, s.config.DialBusyInterval); err != nil {
						// log
					}
				}
			} else {
				// either worked or failed for a reason different than 'too many peers'
				if contains {
					if err := s.dispatcher.Remove(task); err != nil {
						// log
					}
				}
			}

			if err == nil {
				// update the peerstore
				s.peerStore.Update(task, 0)
			}

		case <-s.closeCh:
			return
		}
	}
}

func (s *Server) dialRunner() {
	s.dispatcher.SetEnabled(true)

	tasks := make(chan string, s.config.DialTasks)

	// run the dialtasks
	for i := 0; i < s.config.DialTasks; i++ {
		go s.dialTask(strconv.Itoa(i), tasks)
	}

	sendToTask := func(enode string) {
		tasks <- enode
	}

	for {
		select {
		case enode := <-s.addPeer:
			sendToTask(enode)

		case enode := <-s.Discovery.Deliver():
			sendToTask(enode)

		case enode := <-s.dispatcher.Events():
			sendToTask(enode.ID())

		case <-s.closeCh:
			return
		}
	}
}

// Dial dials an enode (async)
func (s *Server) Dial(enode string) {
	select {
	case s.addPeer <- enode:
	default:
	}
}

// DialSync dials and waits for the result
func (s *Server) DialSync(enode string) error {
	return s.connectWithEnode(enode)
}

// GetPeerByPrefix searches a peer by his prefix
func (s *Server) GetPeerByPrefix(search string) (*Peer, bool) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	for id, peer := range s.peers {
		if strings.HasPrefix(id, search) {
			return peer, true
		}
	}
	return nil, false
}

func (s *Server) GetPeer(id string) *Peer {
	for x, i := range s.peers {
		if id == x {
			return i
		}
	}
	return nil
}

func (s *Server) removePeer(peer *Peer) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	delete(s.peers, peer.ID)
}

func (s *Server) Disconnect() {
	// disconnect the peers
	for _, p := range s.peers {
		p.Close()
	}
}

func (s *Server) connect(addrs string) error {
	return s.connectWithEnode(addrs)
}

func (s *Server) connectWithEnode(rawURL string) error {
	if _, ok := s.peers[rawURL]; ok {
		// TODO: add tests
		// Trying to connect with an already connected id
		// TODO, after disconnect do we remove the peer from this list?
		return nil
	}

	session, err := s.transport.DialTimeout(rawURL, defaultDialTimeout)
	if err != nil {
		return err
	}

	// match protocols
	return s.addSession(session)
}

func (s *Server) addSession(session Session) error {
	p := newPeer(session)

	instances := []*Instance{}
	var instanceLock sync.Mutex

	streams := session.Streams()
	errs := make(chan error, len(streams))

	for _, stream := range streams {
		go func(stream Stream) {
			spec := stream.Protocol()

			proto, ok := s.getProtocol(spec.Name, spec.Version)
			if !ok {
				// This should not happen, its an internal error
				errs <- fmt.Errorf("protocol does not exists")
				return
			}

			if err := proto.HandlerFn(stream, p); err != nil {
				errs <- err
				return
			}

			instanceLock.Lock()
			instances = append(instances, &Instance{
				Protocol: proto,
			})
			instanceLock.Unlock()
			errs <- nil
		}(stream)
	}

	for i := 0; i < len(streams); i++ {
		if err := <-errs; err != nil {
			p.Close()
			return err
		}
	}

	p.protocols = instances

	// Remove peer from list if the session is closed
	go func() {
		<-session.CloseChan()

		s.peersLock.Lock()
		delete(s.peers, p.ID)
		s.peersLock.Unlock()
	}()

	s.peersLock.Lock()
	s.peers[p.ID] = p
	s.peersLock.Unlock()

	select {
	case s.EventCh <- MemberEvent{Type: NodeJoin, Peer: p}:
	default:
	}

	return nil
}

func (s *Server) ID() enode.ID {
	return s.Enode.ID
}

func (s *Server) getProtocol(name string, version uint) (*Protocol, bool) {
	for _, p := range s.config.Protocols {
		proto := p.Spec
		if proto.Name == name && proto.Version == version {
			return p, true
		}
	}
	return nil, false
}

func (s *Server) Close() {
	// close peers
	for _, i := range s.peers {
		i.Close()
	}

	if err := s.peerStore.Close(); err != nil {
		panic(err)
	}

	// close transport
	if err := s.transport.Close(); err != nil {
		s.logger.Printf("[ERROR] failed to close transport: err, %v", err.Error())
	}
}
