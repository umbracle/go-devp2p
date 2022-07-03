package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyEncoding(t *testing.T) {
	for i := 0; i < 10; i++ {
		priv, _ := GenerateKey()

		// marshall private key
		buf, err := MarshallPrivateKey(priv)
		assert.NoError(t, err)

		priv0, err := ParsePrivateKey(buf)
		assert.NoError(t, err)

		assert.Equal(t, priv, priv0)

		// marshall public key
		buf = MarshallPublicKey(&priv.PublicKey)

		pub0, err := ParsePublicKey(buf)
		assert.NoError(t, err)

		assert.Equal(t, priv.PublicKey, *pub0)
	}
}
