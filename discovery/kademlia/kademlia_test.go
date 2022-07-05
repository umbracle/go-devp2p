package kademlia

import (
	"encoding/hex"
	"io"
	"math/rand"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"
)

func hashit(s string) []byte {
	hash := sha3.New256()
	hash.Write([]byte(s))
	return hash.Sum(nil)
}

func convertPeerID(s string) []byte {
	return hashit(s)
}

func randEntry() *Entry {
	id := randPeerID()
	return &Entry{
		id:   id,
		hash: hashit(id),
	}
}

func randPeerID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	buf := make([]byte, 16)
	if _, err := io.ReadFull(r, buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

func TestKademlia_Callbacks(t *testing.T) {
	local := randPeerID()
	rt := NewRoutingTable(10, local, time.Hour, sha3.New256())

	peers := make([]string, 100)
	for i := 0; i < 100; i++ {
		peers[i] = randPeerID()
	}

	pset := make(map[string]struct{})
	rt.PeerAdded = func(p string) {
		pset[p] = struct{}{}
	}
	rt.PeerRemoved = func(p string) {
		delete(pset, p)
	}

	rt.Update(peers[0])
	if _, ok := pset[peers[0]]; !ok {
		t.Fatal("should have this peer")
	}

	rt.Remove(peers[0])
	if _, ok := pset[peers[0]]; ok {
		t.Fatal("should not have this peer")
	}

	for _, p := range peers {
		rt.Update(p)
	}

	out := rt.ListPeers()
	for _, outp := range out {
		if _, ok := pset[outp]; !ok {
			t.Fatal("should have peer in the peerset")
		}
		delete(pset, outp)
	}

	if len(pset) > 0 {
		t.Fatal("have peers in peerset that were not in the table", len(pset))
	}
}

// Right now, this just makes sure that it doesnt hang or crash
func TestKademlia_UpdatePeers(t *testing.T) {
	local := randPeerID()
	rt := NewRoutingTable(10, local, time.Hour, sha3.New256())

	peers := make([]string, 100)
	for i := 0; i < 100; i++ {
		peers[i] = randPeerID()
	}

	// Testing Update
	for i := 0; i < 10000; i++ {
		rt.Update(peers[rand.Intn(len(peers))])
	}

	for i := 0; i < 100; i++ {
		id := randPeerID()
		ret := rt.NearestPeers(id, 5)
		if len(ret) == 0 {
			t.Fatal("Failed to find node near ID.")
		}
	}
}

func TestKademlia_FindNearestPeer(t *testing.T) {
	local := randPeerID()
	rt := NewRoutingTable(10, local, time.Hour, sha3.New256())

	peers := make([]string, 100)
	for i := 0; i < 5; i++ {
		peers[i] = randPeerID()
		rt.Update(peers[i])
	}
	found := rt.NearestPeer(peers[2])
	if !(found == peers[2]) {
		t.Fatalf("Failed to lookup known node...")
	}
}

func TestKademlia_EldestPreferred(t *testing.T) {
	local := randPeerID()
	rt := NewRoutingTable(10, local, time.Hour, sha3.New256())

	// generate size + 1 peers to saturate a bucket
	peers := make([]string, 15)
	for i := 0; i < 15; {
		if p := randPeerID(); CommonPrefixLen(convertPeerID(local), convertPeerID(p)) == 0 {
			peers[i] = p
			i++
		}
	}

	// test 10 first peers are accepted.
	for _, p := range peers[:10] {
		if _, err := rt.Update(p); err != nil {
			t.Errorf("expected all 10 peers to be accepted; instead got: %v", err)
		}
	}

	// test next 5 peers are rejected.
	for _, p := range peers[10:] {
		if _, err := rt.Update(p); err != ErrPeerRejectedNoCapacity {
			t.Errorf("expected extra 5 peers to be rejected; instead got: %v", err)
		}
	}
}

func TestKademlia_FindMultiple(t *testing.T) {
	local := randPeerID()
	rt := NewRoutingTable(20, local, time.Hour, sha3.New256())

	peers := make([]string, 100)
	for i := 0; i < 18; i++ {
		peers[i] = randPeerID()
		rt.Update(peers[i])
	}
	found := rt.NearestPeers(peers[2], 15)
	if len(found) != 15 {
		t.Fatalf("Got back different number of peers than we expected.")
	}
}

// Looks for race conditions in table operations. For a more 'certain'
// test, increase the loop counter from 1000 to a much higher number
// and set GOMAXPROCS above 1
func TestKademlia_Multithreaded(t *testing.T) {
	tab := NewRoutingTable(20, "localPeer", time.Hour, sha3.New256())
	var peers []string
	for i := 0; i < 500; i++ {
		peers = append(peers, randPeerID())
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Update(peers[n])
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Update(peers[n])
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Find(peers[n])
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	<-done
}

func BenchmarkUpdates(b *testing.B) {
	b.StopTimer()
	tab := NewRoutingTable(20, "localKey", time.Hour, sha3.New256())

	var peers []string
	for i := 0; i < b.N; i++ {
		peers = append(peers, randPeerID())
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tab.Update(peers[i])
	}
}

func BenchmarkFinds(b *testing.B) {
	b.StopTimer()
	tab := NewRoutingTable(20, "localKey", time.Hour, sha3.New256())

	var peers []string
	for i := 0; i < b.N; i++ {
		peers = append(peers, randPeerID())
		tab.Update(peers[i])
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tab.Find(peers[i])
	}
}
