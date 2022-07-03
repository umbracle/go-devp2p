package devp2p

import (
	"io/ioutil"
	"log"
	"time"
)

// Config is the p2p server configuration
type Config struct {
	Logger           *log.Logger
	Name             string
	BindAddress      string
	BindPort         int
	MaxPeers         int
	Bootnodes        []string
	DialTasks        int
	DialBusyInterval time.Duration
	PeerStore        PeerStore
	Protocols        []*Protocol
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	c := &Config{
		Name:             "minimal/go1.10.2",
		Logger:           log.New(ioutil.Discard, "", 0),
		BindAddress:      "127.0.0.1",
		BindPort:         30304,
		MaxPeers:         10,
		Bootnodes:        []string{},
		DialTasks:        defaultDialTasks,
		DialBusyInterval: 1 * time.Minute,
		PeerStore:        &NoopPeerStore{},
		Protocols:        []*Protocol{},
	}
	return c
}

type ConfigOption func(*Config)

func WithName(name string) ConfigOption {
	return func(c *Config) {
		c.Name = name
	}
}

func WithBindAddress(addr string) ConfigOption {
	return func(c *Config) {
		c.BindAddress = addr
	}
}

func WithBindPort(port int) ConfigOption {
	return func(c *Config) {
		c.BindPort = port
	}
}

func WithMaxPeers(maxPeers int) ConfigOption {
	return func(c *Config) {
		c.MaxPeers = maxPeers
	}
}

func WithBootnodes(bootnodes []string) ConfigOption {
	return func(c *Config) {
		c.Bootnodes = bootnodes
	}
}

func WithPeerStore(peerstore PeerStore) ConfigOption {
	return func(c *Config) {
		c.PeerStore = peerstore
	}
}

func WithLogger(logger *log.Logger) ConfigOption {
	return func(c *Config) {
		c.Logger = logger
	}
}

func WithProtocol(p *Protocol) ConfigOption {
	return func(c *Config) {
		c.Protocols = append(c.Protocols, p)
	}
}
