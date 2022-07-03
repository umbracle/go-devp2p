package rlpx

import (
	"crypto/rand"
	"testing"

	"github.com/umbracle/go-devp2p"
)

func BenchmarkHandshake(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pipe(nil)
	}
}

func BenchmarkSendRecv256(b *testing.B) {
	benchmarkSendRecv(b, 256)
}

func BenchmarkSendRecv512(b *testing.B) {
	benchmarkSendRecv(b, 512)
}

func BenchmarkSendRecv1024(b *testing.B) {
	benchmarkSendRecv(b, 1024)
}

func BenchmarkSendRecv2048(b *testing.B) {
	benchmarkSendRecv(b, 2048)
}

func BenchmarkSendRecv4096(b *testing.B) {
	benchmarkSendRecv(b, 4096)
}

func BenchmarkSendRecv1MB(b *testing.B) {
	benchmarkSendRecv(b, 1024*1024)
}

func BenchmarkSendRecv2MB(b *testing.B) {
	benchmarkSendRecv(b, 1024*1024*2)
}

func BenchmarkSendRecvMax(b *testing.B) {
	benchmarkSendRecv(b, maxUint24-773)
}

func benchmarkSendRecv(b *testing.B, sendSize uint32) {
	client, server := pipe(nil)
	client.SetSnappy()
	server.SetSnappy()

	spec := devp2p.ProtocolSpec{}
	c := client.OpenStream(20, 10, spec)
	s := server.OpenStream(20, 10, spec)

	sendBuf := make([]byte, sendSize)

	sendHeader := make(Header, HeaderSize)
	sendHeader.Encode(1, uint32(len(sendBuf)))

	recvBuf := make([]byte, sendSize+HeaderSize)
	doneCh := make(chan struct{})

	rand.Read(sendBuf)

	b.SetBytes(int64(sendSize))
	b.ReportAllocs()
	b.ResetTimer()

	go func() {
		for i := 0; i < b.N; i++ {
			if _, err := c.read(recvBuf); err != nil {
				b.Fatal(err)
			}
		}
		doneCh <- struct{}{}
	}()

	for i := 0; i < b.N; i++ {
		if _, err := s.Write(sendHeader); err != nil {
			b.Fatal("")
		}
		if _, err := s.Write(sendBuf); err != nil {
			b.Fatal("")
		}
	}
	<-doneCh
}
