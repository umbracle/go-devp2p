package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/umbracle/ecies"
	"golang.org/x/crypto/sha3"
)

func init() {
	ecies.AddParamsForCurve(S256, ecies.ECIES_AES128_SHA256)
}

// S256 is the secp256k1 elliptic curve
var S256 = btcec.S256()

func ParsePrivateKey(buf []byte) (*ecdsa.PrivateKey, error) {
	prv, _ := btcec.PrivKeyFromBytes(S256, buf)
	return prv.ToECDSA(), nil
}

func MarshallPrivateKey(priv *ecdsa.PrivateKey) ([]byte, error) {
	return (*btcec.PrivateKey)(priv).Serialize(), nil
}

// GenerateKey generates a new key based on the secp256k1 elliptic curve.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256, rand.Reader)
}

// ParsePublicKey parses bytes into a public key on the secp256k1 elliptic curve.
func ParsePublicKey(buf []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(S256, buf)
	if x == nil || y == nil {
		return nil, fmt.Errorf("cannot unmarshall")
	}
	return &ecdsa.PublicKey{Curve: S256, X: x, Y: y}, nil
}

// MarshallPublicKey marshalls a public key on the secp256k1 elliptic curve.
func MarshallPublicKey(pub *ecdsa.PublicKey) []byte {
	return elliptic.Marshal(S256, pub.X, pub.Y)
}

func Ecrecover(hash, sig []byte) ([]byte, error) {
	pub, err := RecoverPubkey(sig, hash)
	if err != nil {
		return nil, err
	}
	return MarshallPublicKey(pub), nil
}

// RecoverPubkey verifies the compact signature "signature" of "hash" for the
// secp256k1 curve.
func RecoverPubkey(signature, hash []byte) (*ecdsa.PublicKey, error) {
	size := len(signature)
	term := byte(27)
	if signature[size-1] == 1 {
		term = 28
	}

	sig := append([]byte{term}, signature[:size-1]...)
	pub, _, err := btcec.RecoverCompact(S256, sig, hash)
	if err != nil {
		return nil, err
	}
	return pub.ToECDSA(), nil
}

// Sign produces a compact signature of the data in hash with the given
// private key on the secp256k1 curve.
func Sign(priv *ecdsa.PrivateKey, hash []byte) ([]byte, error) {
	sig, err := btcec.SignCompact(S256, (*btcec.PrivateKey)(priv), hash, false)
	if err != nil {
		return nil, err
	}
	term := byte(0)
	if sig[0] == 28 {
		term = 1
	}
	return append(sig, term)[1:], nil
}

// Keccak256 calculates the Keccak256
func Keccak256(v ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, i := range v {
		h.Write(i)
	}
	return h.Sum(nil)
}

func CompressPubKey(pub *ecdsa.PublicKey) []byte {
	return (*btcec.PublicKey)(pub).SerializeCompressed()
}

func ParseCompressedPubKey(d []byte) (*ecdsa.PublicKey, error) {
	key, err := btcec.ParsePubKey(d, btcec.S256())
	if err != nil {
		return nil, err
	}
	return key.ToECDSA(), nil
}
