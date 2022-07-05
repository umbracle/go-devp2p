package forkid

import (
	"encoding/hex"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

type chainConfig struct {
	genesis [32]byte
	forks   []uint64
}

var goerliConfig = chainConfig{
	genesis: [32]byte{
		// 0xbf7e331f7f7c1dd2e05159666b3bf8bc7a8a3a9eb1d518969eab529dd9b88c1a
		191, 126, 51, 31, 127, 124, 29, 210, 224, 81, 89, 102, 107, 59, 248, 188, 122, 138, 58, 158, 177, 213, 24, 150, 158, 171, 82, 157, 217, 184, 140, 26,
	},
	forks: []uint64{
		1561651, 4460644, 5062605,
	},
}

var mainnetConfig = chainConfig{
	genesis: [32]byte{
		// 0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3
		212, 229, 103, 64, 248, 118, 174, 248, 192, 16, 184, 106, 64, 213, 245, 103, 69, 161, 24, 208, 144, 106, 52, 230, 154, 236, 140, 13, 177, 203, 143, 163,
	},
	forks: []uint64{
		1150000, 1920000, 2463000, 2675000, 4370000, 7280000, 9069000, 9200000, 12244000, 12965000, 13773000, 15050000,
	},
}

func TestForkID_Create(t *testing.T) {
	type testcase struct {
		head uint64
		hash string
		next uint64
	}

	testFork := func(config chainConfig, cases []testcase) {
		forkid := NewForkID(config.genesis, config.forks)

		for _, tt := range cases {
			id, err := hex.DecodeString(tt.hash)
			assert.NoError(t, err)

			idFound, next := forkid.At(tt.head)
			assert.Equal(t, id, idFound[:])
			assert.Equal(t, next, tt.next)
		}
	}

	// mainnet
	testFork(mainnetConfig, []testcase{
		{0, "fc64ec04", 1150000},
		{1149999, "fc64ec04", 1150000},
		{1150000, "97c2c34c", 1920000},
		{1919999, "97c2c34c", 1920000},
		{1920000, "91d1f948", 2463000},
		{2462999, "91d1f948", 2463000},
		{2463000, "7a64da13", 2675000},
		{2674999, "7a64da13", 2675000},
		{2675000, "3edd5b10", 4370000},
		{4369999, "3edd5b10", 4370000},
		{4370000, "a00bc324", 7280000},
		{7279999, "a00bc324", 7280000},
		{7280000, "668db0af", 9069000},
		{7987396, "668db0af", 9069000},
	})

	// goerli
	testFork(goerliConfig, []testcase{
		{0, "a3f5ab08", 1561651},
		{1561650, "a3f5ab08", 1561651},
		{1561651, "c25efa5c", 4460644},
		{2000000, "c25efa5c", 4460644},
	})
}

func TestForkID_Validate(t *testing.T) {
	var cases = []struct {
		head uint64
		id   string
		next uint64
		err  error
	}{
		{7987396, "668db0af", 0, nil},
		{7987396, "668db0af", math.MaxUint64, nil},
		{7279999, "a00bc324", 0, nil},
		{7279999, "a00bc324", 7280000, nil},
		{7279999, "a00bc324", math.MaxUint64, nil},
		{7987396, "a00bc324", 7280000, nil},

		{7987396, "3edd5b10", 4370000, nil},
		{7279999, "668db0af", 0, nil},
		{4369999, "a00bc324", 0, nil},
		{7987396, "a00bc324", 0, ErrRemoteStale},
		{7987396, "5cddc0e1", 0, ErrLocalIncompatibleOrStale},
		{7279999, "5cddc0e1", 0, ErrLocalIncompatibleOrStale},
		{7987396, "afec6b27", 0, ErrLocalIncompatibleOrStale},
		{88888888, "668db0af", 88888888, ErrRemoteStale},
		{7279999, "a00bc324", 7279999, ErrLocalIncompatibleOrStale},
	}

	forkid := NewForkID(mainnetConfig.genesis, mainnetConfig.forks)
	for _, c := range cases {
		id, err := hex.DecodeString(c.id)
		assert.NoError(t, err)

		err = forkid.Validate(c.head, id, c.next)
		assert.Equal(t, c.err, err)
	}
}

func TestForkID_Encoding(t *testing.T) {

}
