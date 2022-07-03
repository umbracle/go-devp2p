package discovery

import (
	"log"
	"net"
	"sync/atomic"
	"time"
)

// UDPTransport implements the UDP Transport
type UDPTransport struct {
	addr     *net.UDPAddr
	logger   log.Logger
	packetCh chan *Packet
	listener *net.UDPConn
	shutdown int32
}

func newUDPTransport(udpAddr *net.UDPAddr) (Transport, error) {
	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, nil
	}

	t := &UDPTransport{
		addr:     udpAddr,
		listener: listener,
		packetCh: make(chan *Packet),
	}
	go t.listen()
	return t, nil
}

func (u *UDPTransport) listen() {
	for {
		buf := make([]byte, udpPacketBufSize)

		n, addr, err := u.listener.ReadFrom(buf)
		ts := time.Now()
		if err != nil {
			if s := atomic.LoadInt32(&u.shutdown); s == 1 {
				break
			}
			u.logger.Printf("[ERROR] Error reading UDP packet: %v", err)
			continue
		}
		if n < 1 {
			u.logger.Printf("[ERROR] UDP packet too short: len, %d, addr, %s", len(buf), addr)
			continue
		}

		u.packetCh <- &Packet{
			Buf:       buf[:n],
			From:      addr,
			Timestamp: ts,
		}
	}
}

// Addr implements the transport interface
func (u *UDPTransport) Addr() *net.UDPAddr {
	return u.addr
}

// PacketCh implements the transport interface
func (u *UDPTransport) PacketCh() chan *Packet {
	return u.packetCh
}

// WriteTo implements the transport interface
func (u *UDPTransport) WriteTo(b []byte, addr string) (time.Time, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return time.Time{}, err
	}
	_, err = u.listener.WriteTo(b, udpAddr)
	return time.Now(), err
}

// Shutdown implements the transport interface
func (u *UDPTransport) Shutdown() {
	atomic.StoreInt32(&u.shutdown, 1)
	u.listener.Close()
}
