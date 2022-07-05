package kademlia

import (
	"math/rand"
	"testing"
)

// Test basic features of the bucket struct
func TestBucket(t *testing.T) {
	b := newBucket()

	peers := make([]*Entry, 100)
	for i := 0; i < 100; i++ {
		peers[i] = randEntry()
		b.PushFront(peers[i])
	}

	local := randPeerID()
	localID := convertPeerID(local)

	i := rand.Intn(len(peers))
	if !b.Has(peers[i].id) {
		t.Errorf("Failed to find peer: %v", peers[i])
	}

	spl := b.Split(0, convertPeerID(local))
	llist := b.list
	for e := llist.Front(); e != nil; e = e.Next() {
		p := e.Value.(*Entry).hash
		cpl := CommonPrefixLen(p, localID)
		if cpl > 0 {
			t.Fatalf("Split failed. found id with cpl > 0 in 0 bucket")
		}
	}

	rlist := spl.list
	for e := rlist.Front(); e != nil; e = e.Next() {
		p := e.Value.(*Entry).hash
		cpl := CommonPrefixLen(p, localID)
		if cpl == 0 {
			t.Fatalf("Split failed. found id with cpl == 0 in non 0 bucket")
		}
	}
}
