package eth

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/umbracle/fastrlp"
	"github.com/umbracle/go-devp2p"
)

type Eth66Backend interface {
	Status() *Status
	NotifyPeer(p *Peer)
	GetBlockHeader(req *BlockHeadersPacket) Eth66Body
	GetBlockBodies(hashes [][32]byte) Eth66Body
	GetTransactions(hashes [][32]byte) Eth66Body
	NotifyTransactionHashes(hashes [][32]byte)
	NotifyTransactions(v *fastrlp.Value)
	NotifyBlockHashes(hashes []*NewBlockHash)
}

type Eth66Protocol struct {
	Impl Eth66Backend
}

type Peer struct {
	handler *handler
	closeCh chan struct{}
}

func (p *Peer) request(dst RlpResponse, code ethMessage, msg Eth66Body) error {
	req := &Request{
		RequestId: rand.Uint64(),
		Body:      msg,
	}
	if err := p.handler.Write(code, req); err != nil {
		return err
	}
	return p.handler.doRequest(dst, req)
}

func (p *Peer) GetTransactions(dst RlpResponse, hashes [][32]byte) error {
	obj := HashList(hashes)
	return p.request(dst, GetPooledTransactionsMsg, &obj)
}

func (p *Peer) GetBlockByNumber(dst RlpResponse, i uint64) error {
	obj := &BlockHeadersPacket{
		Number: i,
		Amount: 1,
	}
	return p.request(dst, GetBlockHeadersMsg, obj)
}

func (p *Peer) CloseCh() chan struct{} {
	return p.closeCh
}

func (p *Peer) close() {
	close(p.closeCh)
}

// handler is an instance that runs for every peer and handles
// the delivery and management of messages
type handler struct {
	Impl     Eth66Backend
	peer     *devp2p.Peer
	conn     devp2p.Stream
	inflight sync.Map
}

type inflightRequest struct {
	ch   chan error
	resp RlpResponse
}

func (h *handler) doRequest(dst RlpResponse, req *Request) error {
	req2 := &inflightRequest{
		ch:   make(chan error),
		resp: dst,
	}
	h.inflight.Store(req.RequestId, req2)

	defer func() {
		close(req2.ch)
		h.inflight.Delete(req.RequestId)
	}()

	select {
	case err := <-req2.ch:
		return err
	case <-time.After(6 * time.Second):
		return fmt.Errorf("timeout")
	}
}

func (h *handler) deliverResponse(resp *Response) (err error) {
	raw, ok := h.inflight.Load(resp.RequestId)
	if !ok {
		return fmt.Errorf("id not found")
	}
	req := raw.(*inflightRequest)

	defer func() {
		select {
		case req.ch <- err:
		default:
		}
	}()

	if err = req.resp.UnmarshalRLPWith(resp.BodyRaw); err != nil {
		return err
	}
	return nil
}

func (h *handler) run() error {
	localStatus := h.Impl.Status()

	pp := &Peer{
		handler: h,
		closeCh: make(chan struct{}),
	}
	defer pp.close()

	buf, _, err := h.conn.ReadMsg()
	if err != nil {
		fmt.Printf("failed to read msg: %v\n", err)
		return err
	}

	go func() {
		if err := h.Write(StatusMsg, localStatus); err != nil {
			fmt.Printf("[ERROR]: %v\n", err)
		}
	}()

	status := &Status{}
	if err := status.UnmarshalRLP(buf); err != nil {
		panic(err)
	}

	if err := localStatus.Equal(status); err != nil {
		h.Close()
		return err
	}

	fmt.Println("_ GOOD PEER _", h.peer.Info.Client, h.peer.PrettyID(), status, localStatus)

	go func() {
		//time.Sleep(5 * time.Second)
		body := &RlpList{}
		fmt.Println(pp.GetBlockByNumber(body, 0))
	}()

	h.Impl.NotifyPeer(pp)

	for {
		buf, code, err := h.conn.ReadMsg()
		if err != nil {
			return err
		}
		fmt.Printf("Received message %s %d\n", h.peer.PrettyID(), code)

		if err := h.handleMsg(code, buf); err != nil {
			fmt.Printf("[ERROR] handle message %s %v\n", h.peer.PrettyID(), err)
		}
	}
}

func (h *handler) handleMsg(code uint16, buf []byte) error {

	deliverResponse := func() error {
		msg := &Response{}
		if err := msg.UnmarshalRLP(buf); err != nil {
			return err
		}
		if err := h.deliverResponse(msg); err != nil {
			return err
		}
		return nil
	}

	switch ethMessage(code) {
	case TransactionsMsg:
		// done

		p := fastrlp.Parser{}
		v, err := p.Parse(buf)
		if err != nil {
			return err
		}
		h.Impl.NotifyTransactions(v)

	case NewBlockHashesMsg:
		msg := &newBlockHashesPacket{}
		if err := msg.UnmarshalRLP(buf); err != nil {
			return err
		}
		h.Impl.NotifyBlockHashes(*msg)

	case NewPooledTransactionHashesMsg:
		// done
		m := &HashList{}
		if err := m.UnmarshalRLP(buf); err != nil {
			return err
		}
		h.Impl.NotifyTransactionHashes(*m)

	case BlockBodiesMsg:
		if err := deliverResponse(); err != nil {
			return err
		}

	case GetBlockBodiesMsg:

		body := &HashList{}
		msg := &Response{
			Body: body,
		}
		if err := msg.UnmarshalRLP(buf); err != nil {
			return err
		}
		resp := &Request{
			RequestId: msg.RequestId,
			Body:      h.Impl.GetBlockBodies(*body),
		}
		if err := h.Write(BlockBodiesMsg, resp); err != nil {
			return err
		}

	case BlockHeadersMsg:

		// response to block headers
		if err := deliverResponse(); err != nil {
			return err
		}

	case GetBlockHeadersMsg:

		body := &BlockHeadersPacket{}
		msg := &Response{
			Body: body,
		}
		if err := msg.UnmarshalRLP(buf); err != nil {
			return err
		}
		resp := &Request{
			RequestId: msg.RequestId,
			Body:      h.Impl.GetBlockHeader(body),
		}
		if err := h.Write(BlockHeadersMsg, resp); err != nil {
			return err
		}

	case PooledTransactionsMsg:

		if err := deliverResponse(); err != nil {
			return err
		}

	case GetPooledTransactionsMsg:

		body := &HashList{}
		msg := &Response{
			Body: body,
		}
		if err := msg.UnmarshalRLP(buf); err != nil {
			return err
		}
		resp := &Request{
			RequestId: msg.RequestId,
			Body:      h.Impl.GetTransactions(*body),
		}
		if err := h.Write(PooledTransactionsMsg, resp); err != nil {
			return err
		}

	case NewBlockMsg:
		// unhandled

	default:
		panic(fmt.Errorf("message not handled: %d", code))
	}

	return nil
}

func (h *handler) Close() error {
	return h.conn.Close()
}

func (b *Eth66Protocol) Eth66() *devp2p.Protocol {
	return &devp2p.Protocol{
		Spec: devp2p.ProtocolSpec{
			Name:    "eth",
			Version: 66,
			Length:  17,
		},
		HandlerFn: func(conn1 devp2p.Stream, peer *devp2p.Peer) error {
			h := handler{
				conn: conn1,
				peer: peer,
				Impl: b.Impl,
			}
			err := h.run()
			return err
		},
	}
}

type rlpMessage interface {
	MarshalRLP() ([]byte, error)
}

func (h *handler) Write(code ethMessage, msg rlpMessage) error {
	b, err := msg.MarshalRLP()
	if err != nil {
		return err
	}
	return h.conn.WriteMsg(uint64(code), b)
}

type ethMessage int16

const (
	StatusMsg ethMessage = 0x00

	NewBlockHashesMsg             = 0x01
	TransactionsMsg               = 0x02
	NewBlockMsg                   = 0x07
	NewPooledTransactionHashesMsg = 0x08

	GetBlockHeadersMsg = 0x03
	BlockHeadersMsg    = 0x04

	GetBlockBodiesMsg = 0x05
	BlockBodiesMsg    = 0x06

	GetNodeDataMsg = 0x0d
	NodeDataMsg    = 0x0e

	GetReceiptsMsg = 0x0f
	ReceiptsMsg    = 0x10

	GetPooledTransactionsMsg = 0x09
	PooledTransactionsMsg    = 0x0a
)
