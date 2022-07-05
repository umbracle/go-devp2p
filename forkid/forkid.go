package forkid

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"sort"
)

type ID [4]byte

type ForkID struct {
	checksums checksums
}

func NewForkID(genesis [32]byte, forks []uint64) *ForkID {
	var checksums checksums
	forks = cleanForks(forks)

	// build the fork sums
	builder := newCRC32(genesis[:])
	checksums.add(0, builder.Hash())

	for _, fork := range forks {
		checksums.add(fork, builder.Add(fork).Hash())
	}

	// add an extra fork on the infinite
	checksums.add(math.MaxUint64, builder.Add(math.MaxUint64).Hash())

	forkId := &ForkID{
		checksums: checksums,
	}
	return forkId
}

func (f *ForkID) findForkAt(block uint64) int {
	indx := sort.Search(len(f.checksums), func(i int) bool {
		return f.checksums[i].Number > block
	})
	return indx - 1
}

func (f *ForkID) At(block uint64) (ID, uint64) {
	indx := f.findForkAt(block)
	return f.checksums[indx].Checksum, f.checksums[indx+1].Number
}

var (
	ErrRemoteStale              = fmt.Errorf("stalled")
	ErrLocalIncompatibleOrStale = fmt.Errorf("incomptabible or stalled")
)

func (f *ForkID) Validate(localHead uint64, remoteID []byte, remoteNext uint64) error {
	indx := f.findForkAt(localHead)
	localFork := f.checksums[indx].Checksum

	if bytes.Equal(localFork[:], remoteID) {
		// in the same fork, is there any *active* future forks that local is not aware?
		if remoteNext > 0 && localHead >= remoteNext {
			return ErrLocalIncompatibleOrStale
		}
		return nil
	}

	// local and remote are in different forks
	// check if remote fork is in a lower set of the forks
	for i := 0; i < indx; i++ {
		if bytes.Equal(f.checksums[i].Checksum[:], remoteID) {
			// next fork must match
			if remoteNext != f.checksums[i+1].Number {
				return ErrRemoteStale
			}
			return nil
		}
	}

	// check if remote fork is higher set than us
	for i := indx; i < len(f.checksums); i++ {
		if bytes.Equal(f.checksums[i].Checksum[:], remoteID) {
			return nil
		}
	}
	return ErrLocalIncompatibleOrStale
}

func cleanForks(forks []uint64) []uint64 {
	// sort the forks
	sort.Slice(forks, func(i, j int) bool {
		return forks[i] < forks[j]
	})

	// remove repeated items
	j := 1
	for i := 1; i < len(forks); i++ {
		if forks[i] != forks[i-1] {
			forks[j] = forks[i]
			j++
		}
	}
	forks = forks[0:j]

	// skip block 0 genesis
	for j = 0; j < len(forks); j++ {
		if forks[j] != 0 {
			break
		}
	}
	return forks[j:]
}

type checksums []checksum

func (c *checksums) add(fork uint64, hash ID) {
	*c = append(*c, checksum{fork, hash})
}

type checksum struct {
	Number   uint64
	Checksum ID
}

type crc32Checksum struct {
	hash uint32
}

func newCRC32(genesis []byte) *crc32Checksum {
	checksum := &crc32Checksum{
		hash: crc32.ChecksumIEEE(genesis[:]),
	}
	return checksum
}

func (c *crc32Checksum) Add(fork uint64) *crc32Checksum {
	var forkBytes [8]byte
	binary.BigEndian.PutUint64(forkBytes[:], fork)

	c.hash = crc32.Update(c.hash, crc32.IEEETable, forkBytes[:])
	return c
}

func (c *crc32Checksum) Hash() [4]byte {
	var blob [4]byte
	binary.BigEndian.PutUint32(blob[:], c.hash)
	return blob
}
