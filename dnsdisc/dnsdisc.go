package dnsdisc

import (
	"context"
	"net"

	"github.com/umbracle/go-devp2p/enr"
)

type DnsDisc struct {
	dns string

	root     *entryRoot
	resolver *net.Resolver

	missing []string
	visited map[string]struct{}

	current *enr.Record
}

func (d *DnsDisc) nextNode() (*enr.Record, error) {
	if d.missing == nil {
		d.missing = []string{}
	}
	if d.visited == nil {
		d.visited = map[string]struct{}{}
	}

	if d.root == nil {
		// resolve entry root
		d.resolver = new(net.Resolver)

		res, err := d.resolver.LookupTXT(context.Background(), d.dns)
		if err != nil {
			return nil, err
		}
		entryRoot, err := parseEntryRoot(res[0])
		if err != nil {
			return nil, err
		}
		d.root = entryRoot
		d.missing = []string{entryRoot.eroot}
	}

	for {
		if len(d.missing) == 0 {
			return nil, nil
		}

		target := d.missing[0]
		d.missing = d.missing[1:]

		if _, ok := d.visited[target]; ok {
			continue
		}
		d.visited[target] = struct{}{}

		data, err := d.resolver.LookupTXT(context.Background(), target+"."+d.dns)
		if err != nil {
			return nil, err
		}
		for _, i := range data {
			res, err := parseEntry(i)
			if err != nil {
				return nil, err
			}
			switch obj := res.(type) {
			case *entryBranch:
				// DPS
				d.missing = append(obj.hashes, d.missing...)

			case *enr.Record:
				return obj, nil
			}
		}
	}
}

func (d *DnsDisc) Has() bool {
	current, err := d.nextNode()
	if err != nil {
		// LOG
	}
	d.current = current
	return d.current != nil
}

func (d *DnsDisc) Next() *enr.Record {
	return d.current
}
