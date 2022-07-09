package dnsdisc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO

func TestDnsDisc(t *testing.T) {

	resolver := &localResolver{
		entries: map[string]string{
			"d":                            "enrtree-root:v1 e=JWXYDBPXYWG6FX3GMDIBFA6CJ4 l=C7HRFPF3BLGF3YR4DY5KX3SMBE seq=1 sig=o908WmNp7LibOfPsr4btQwatZJ5URBr2ZAuxvK4UWHlsB9sUOTJQaGAlLPVAhM__XJesCHxLISo94z5Z2a463gA",
			"C7HRFPF3BLGF3YR4DY5KX3SMBE.d": "",
			"JWXYDBPXYWG6FX3GMDIBFA6CJ4.d": "enrtree-branch:2XS2367YHAXJFGLZHVAWLQD4ZY,H4FHT4B454P6UXFD7JCYQ5PWDY,MHTDO6TMUBRIA2XWG5LUDACK24",
			"2XS2367YHAXJFGLZHVAWLQD4ZY.d": "enr:-HW4QOFzoVLaFJnNhbgMoDXPnOvcdVuj7pDpqRvh6BRDO68aVi5ZcjB3vzQRZH2IcLBGHzo8uUN3snqmgTiE56CH3AMBgmlkgnY0iXNlY3AyNTZrMaECC2_24YYkYHEgdzxlSNKQEnHhuNAbNlMlWJxrJxbAFvA",
			"H4FHT4B454P6UXFD7JCYQ5PWDY.d": "enr:-HW4QAggRauloj2SDLtIHN1XBkvhFZ1vtf1raYQp9TBW2RD5EEawDzbtSmlXUfnaHcvwOizhVYLtr7e6vw7NAf6mTuoCgmlkgnY0iXNlY3AyNTZrMaECjrXI8TLNXU0f8cthpAMxEshUyQlK-AM0PW2wfrnacNI",
			"MHTDO6TMUBRIA2XWG5LUDACK24.d": "enr:-HW4QLAYqmrwllBEnzWWs7I5Ev2IAs7x_dZlbYdRdMUx5EyKHDXp7AV5CkuPGUPdvbv1_Ms1CPfhcGCvSElSosZmyoqAgmlkgnY0iXNlY3AyNTZrMaECriawHKWdDRk2xeZkrOXBQ0dfMFLHY4eENZwdufn1S1o",
		},
	}
	d := NewDnsDiscovery("d")
	d.resolver = resolver

	count := 0
	for d.Has() {
		count++
	}
	assert.Equal(t, count, 3)
}
