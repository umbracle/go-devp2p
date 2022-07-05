package dnsdisc

import "testing"

func TestDnsDisc_XX(t *testing.T) {
	d := &DnsDisc{
		dns: "all.goerli.ethdisco.net",
	}
	d.run()
}
