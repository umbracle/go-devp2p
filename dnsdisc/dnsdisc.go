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

	//missing := []string{
	//	entryRoot.eroot,
	//}

	var resolve func(target string)
	resolve = func(target string) {
		fmt.Println(target)

		data, err := resolver.LookupTXT(context.Background(), target+"."+d.dns)
		if err != nil {
			panic(err)
		}
		for _, i := range data {
			fmt.Println(i)
			entries, err := parseBranchRoot(i)
			if err != nil {
				panic(err)
			}

			for _, entry := range entries.hashes {
				resolve(entry)
			}
			//missing = append(missing, entry.hashes...)
			//fmt.Println(entry.hashes)
		}
	}

	resolve(entryRoot.eroot)

	/*
		for {
			if len(missing) == 0 {
				break
			}

			target := missing[0]
			missing = missing[1:]

			fmt.Println("----")
			fmt.Println(target)

			data, err := resolver.LookupTXT(context.Background(), target+"."+d.dns)
			if err != nil {
				panic(err)
			}
			for _, i := range data {
				entry, err := parseBranchRoot(i)
				if err != nil {
					panic(err)
				}
				missing = append(missing, entry.hashes...)
				fmt.Println(entry.hashes)
			}
		}
	*/
}
