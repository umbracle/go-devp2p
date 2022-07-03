package discovery

import (
	"context"
	"crypto/ecdsa"
	"log"

	"github.com/umbracle/go-devp2p/enode"
)

// Discovery interface must be implemented for a discovery protocol
type Discovery interface {
	// Close closes the backend
	Close() error

	// Deliver returns discovered elements
	Deliver() chan string

	// Schedule starts the discovery
	Schedule()
}

// DiscoveryConfig contains configuration parameters
type DiscoveryConfig struct {
	// Logger to be used by the backend
	Logger *log.Logger

	// Enode is the identification of the node
	Enode *enode.Enode

	// Private key of the node to encrypt/decrypt messages
	Key *ecdsa.PrivateKey

	Bootnodes []string
}

type Factory func(context.Context, *DiscoveryConfig) (Discovery, error)
