package dnsdisc

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/umbracle/ethgo"
	"github.com/umbracle/go-devp2p/enr"
)

// List of dns discovery domains https://github.com/ethereum/discv4-dns-lists

type DnsDisc struct {
	logger *log.Logger
	dns    string

	root     *entryRoot
	resolver Resolver

	missing []string
	current *enr.Record
}

func NewDnsDiscovery(dnsRoot string) *DnsDisc {
	disc := &DnsDisc{
		resolver: new(net.Resolver),
		missing:  []string{},
		dns:      dnsRoot,
		logger:   log.New(ioutil.Discard, "", 0),
	}
	return disc
}

func (d *DnsDisc) SetLogger(logger *log.Logger) {
	d.logger = logger
}

func (d *DnsDisc) resolveRoot() error {
	// resolve entry root
	res, err := d.resolver.LookupTXT(context.Background(), d.dns)
	if err != nil {
		return err
	}
	entryRoot, err := parseEntryRoot(res[0])
	if err != nil {
		return err
	}
	d.root = entryRoot
	d.missing = []string{entryRoot.eroot}

	return nil
}

func (d *DnsDisc) nextNode() (*enr.Record, error) {
	if d.root == nil {
		// resolve entry root
		if err := d.resolveRoot(); err != nil {
			return nil, err
		}
	}

	for {
		if len(d.missing) == 0 {
			return nil, nil
		}

		target := d.missing[0]
		d.missing = d.missing[1:]

		data, err := d.resolver.LookupTXT(context.Background(), target+"."+d.dns)
		if err != nil {
			return nil, err
		}
		expectedPrefix, err := base32.DecodeString(target)
		if err != nil {
			return nil, err
		}

		for _, i := range data {
			res, err := parseEntry(i)
			if err != nil {
				return nil, err
			}

			txtHash := ethgo.Keccak256([]byte(i))
			if !bytes.HasPrefix(txtHash, expectedPrefix) {
				return nil, fmt.Errorf("incorrect hash")
			}

			switch obj := res.(type) {
			case *entryBranch:
				// DPS
				d.missing = append(obj.hashes, d.missing...)

			case *enrEntry:
				return obj.record, nil
			}
		}
	}
}

func (d *DnsDisc) Has() bool {
	current, err := d.nextNode()
	if err != nil {
		d.logger.Printf("[ERROR]: %v", err)
	}
	d.current = current
	return d.current != nil
}

func (d *DnsDisc) Next() *enr.Record {
	return d.current
}

type Resolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

type localResolver struct {
	entries map[string]string
}

func (l *localResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	v, ok := l.entries[name]
	if ok {
		return []string{v}, nil
	}
	return nil, fmt.Errorf("entry '%s' not found", name)
}
