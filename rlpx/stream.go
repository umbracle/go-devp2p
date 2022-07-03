package rlpx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/umbracle/go-devp2p"
)

// Stream represents a logic stream within a RLPx session
type Stream struct {
	offset uint64
	length uint64

	conn     *Session
	respLock sync.Mutex

	recvBuf  *bytes.Buffer
	recvLock sync.Mutex

	recvNotifyCh chan struct{}
	errorCh      chan error

	readDeadline  atomic.Value // time.Time
	writeDeadline atomic.Value // time.Time

	header   Header
	protocol devp2p.ProtocolSpec
}

// NewStream constructs a new stream with a given offset and length
func NewStream(offset uint64, length uint64, conn *Session) *Stream {
	s := &Stream{
		offset:       offset,
		length:       length,
		conn:         conn,
		respLock:     sync.Mutex{},
		recvNotifyCh: make(chan struct{}, 1),
		errorCh:      make(chan error, 1),
	}
	s.readDeadline.Store(time.Time{})
	s.writeDeadline.Store(time.Time{})
	return s
}

const (
	sizeOfType   = 2
	sizeOfLength = 4

	// HeaderSize is the size of the RLPx header
	HeaderSize = sizeOfType + sizeOfLength
)

// Header represents an RLPx header
type Header []byte

// MsgType returns the msg code in a RLPx header
func (h Header) MsgType() uint16 {
	return binary.BigEndian.Uint16(h[0:2])
}

// Length returns the length of the data in a RLPx header
func (h Header) Length() uint32 {
	return binary.BigEndian.Uint32(h[2:6])
}

// Encode encodes a header with msg code and length
func (h Header) Encode(msgType uint16, length uint32) {
	binary.BigEndian.PutUint16(h[0:2], msgType)
	binary.BigEndian.PutUint32(h[2:6], length)
}

func (s *Stream) Protocol() devp2p.ProtocolSpec {
	return s.protocol
}

func (s *Stream) WriteMsg(code uint64, b []byte) error {
	if err := s.conn.WriteRawMsg(uint64(code)+s.offset, b); err != nil {
		return err
	}
	return nil
}

// Write implements the net.Conn interface
func (s *Stream) Write(b []byte) (int, error) {
	if s.header == nil {
		if len(b) != HeaderSize {
			return 0, fmt.Errorf("bad length, expected the header")
		}

		s.header = make([]byte, HeaderSize)
		copy(s.header[:], b[:])

		if s.header.Length() == 0 {
			b = nil
		} else {
			return len(b), nil
		}
	}

	size, code := s.header.Length(), s.header.MsgType()
	s.header = nil

	if len(b) != int(size) {
		return 0, fmt.Errorf("expected message of length %d but found %d", size, len(b))
	}
	if err := s.conn.WriteRawMsg(uint64(code)+s.offset, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (s *Stream) ReadMsg() ([]byte, uint16, error) {
	header := make(Header, HeaderSize)

	if _, err := s.read(header[:]); err != nil {
		return nil, 0, err
	}
	buf := make([]byte, header.Length())
	if _, err := s.read(buf); err != nil {
		return nil, 0, err
	}
	return buf, header.MsgType(), nil
}

// read implements the net.Conn interface
func (s *Stream) read(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	defer asyncNotify(s.recvNotifyCh)

START:

	// If there is no data available, block
	s.recvLock.Lock()

	if s.recvBuf == nil || s.recvBuf.Len() == 0 {
		s.recvLock.Unlock()
		goto WAIT
	}

	// Read bytes
	n, _ = s.recvBuf.Read(b)
	s.recvLock.Unlock()

	return n, err

WAIT:
	var timeout <-chan time.Time
	var timer *time.Timer
	readDeadline := s.readDeadline.Load().(time.Time)
	if !readDeadline.IsZero() {
		delay := readDeadline.Sub(time.Now())
		timer = time.NewTimer(delay)
		timeout = timer.C
	}

	select {
	case err := <-s.errorCh:
		return n, err
	case <-s.recvNotifyCh:
		if timer != nil {
			timer.Stop()
		}
		goto START
	case <-timeout:
		return 0, fmt.Errorf("timeout")
	}
}

// asyncNotify is used to signal a waiting goroutine
func asyncNotify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

// readData is used to handle a data frame
func (s *Stream) readData(code uint64, buf []byte) error {
	// Copy into buffer
	s.recvLock.Lock()

	size := len(buf)

	if s.recvBuf == nil {
		s.recvBuf = bytes.NewBuffer(make([]byte, 0, size+HeaderSize))
	}

	h := make(Header, HeaderSize)
	h.Encode(uint16(code-uint64(s.offset)), uint32(size))

	if _, err := s.recvBuf.Write(h); err != nil {
		s.recvLock.Unlock()
		return err
	}

	if _, err := s.recvBuf.Write(buf); err != nil {
		s.recvLock.Unlock()
		return err
	}
	s.recvLock.Unlock()

	// Unblock any readers
	asyncNotify(s.recvNotifyCh)
	return nil
}

// RemoteAddr implements the net.Conn interface
func (s *Stream) RemoteAddr() net.Addr {
	return s.conn.conn.RemoteAddr()
}

// LocalAddr implements the net.Conn interface
func (s *Stream) LocalAddr() net.Addr {
	return s.conn.conn.LocalAddr()
}

// SetDeadline implements the net.Conn interface
func (s *Stream) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

// SetReadDeadline implements the net.Conn interface
func (s *Stream) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

// SetWriteDeadline implements the net.Conn interface
func (s *Stream) SetWriteDeadline(t time.Time) error {
	return s.SetWriteDeadline(t)
}

// Close implements the net.Conn interface
func (s *Stream) Close() error {
	return s.conn.Close()
}

func copyZeroAlloc(w io.Writer, r io.Reader) (int64, error) {
	vbuf := copyBufPool.Get()
	buf := vbuf.([]byte)
	n, err := io.CopyBuffer(w, r, buf)
	copyBufPool.Put(vbuf)
	return n, err
}

var copyBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}
