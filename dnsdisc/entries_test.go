package dnsdisc

import (
	"crypto/ecdsa"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/go-devp2p/crypto"
)

func TestEntries(t *testing.T) {
	newPubKey := func(str string) *ecdsa.PublicKey {
		buf, err := hex.DecodeString(str)
		assert.NoError(t, err)

		pubKey, err := crypto.ParsePublicKey(buf)
		assert.NoError(t, err)

		return pubKey
	}

	cases := []struct {
		str   string
		entry Entry
	}{
		{
			"enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@snap.mainnet.ethdisco.net",
			&entryLink{
				domain: "snap.mainnet.ethdisco.net",
				pubKey: newPubKey("0481b033cb78704a0d956d36195609e807cee3f87b8e9590beb6bd0713959d06f2b0058d44f288666a5000bd4fc5a876788bba09d7c1d49b4b786c4143bf1011d8"),
			},
		},
		{
			"enrtree-branch:BRU43CYW2S4HCEES3DXJ2QYOYQ,AU6OB5RACUZMZMDZJLNJT7TGY4,BVCJRD3VTLDVFX7OPWPMP33SAU",
			&entryBranch{
				hashes: []string{
					"BRU43CYW2S4HCEES3DXJ2QYOYQ", "AU6OB5RACUZMZMDZJLNJT7TGY4", "BVCJRD3VTLDVFX7OPWPMP33SAU",
				},
			},
		},
		{
			"enrtree-root:v1 e=O4E5ES6EIACUASHASBGJGEC67M l=FDXN3SN67NA5DKA4J2GOK7BVQI seq=3189 sig=1SSfIYpZxREoK6eGeJZqicZb87O4y8D8YPOD2omG-C8Sb0aD0yInfMjX3F_GEUNHZKt4bpdQsZSJZ-16pndwtQE",
			&entryRoot{
				eroot: "O4E5ES6EIACUASHASBGJGEC67M",
				lroot: "FDXN3SN67NA5DKA4J2GOK7BVQI",
				sig:   "1SSfIYpZxREoK6eGeJZqicZb87O4y8D8YPOD2omG-C8Sb0aD0yInfMjX3F_GEUNHZKt4bpdQsZSJZ-16pndwtQE",
				seq:   3189,
			},
		},
	}

	for _, c := range cases {
		entry, err := parseEntry(c.str)
		assert.NoError(t, err)
		assert.Equal(t, c.entry, entry)
		assert.Equal(t, c.str, entry.String())
	}
}
