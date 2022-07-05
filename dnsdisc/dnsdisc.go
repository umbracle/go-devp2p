package dnsdisc

import (
	"context"
	"fmt"
	"net"
)

type DnsDisc struct {
	dns string
}

func (d *DnsDisc) Resolve(domain string) {
	resolver := new(net.Resolver)
	fmt.Println(resolver.LookupTXT(context.Background(), "all.goerli.ethdisco.net"))
}

func (d *DnsDisc) run() {
	resolver := new(net.Resolver)

	// resolve root
	res, err := resolver.LookupTXT(context.Background(), d.dns)
	if err != nil {
		panic(err)
	}
	entryRoot, err := parseEntryRoot(res[0])
	if err != nil {
		panic(err)
	}

	fmt.Println("-- entry root --")
	fmt.Println(entryRoot.eroot, entryRoot.lroot)

	data, err := resolver.LookupTXT(context.Background(), entryRoot.eroot+"."+d.dns)
	if err != nil {
		panic(err)
	}
	for _, i := range data {
		parseBranchRoot(i)
	}
}
