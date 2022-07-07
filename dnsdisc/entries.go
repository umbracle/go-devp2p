package dnsdisc

import (
	"fmt"
	"strings"

	"github.com/umbracle/go-devp2p/enr"
)

// entries prefixes
var (
	entryENRPrefix    = "enr:"
	entryRootPrefix   = "enrtree-root:v1"
	entryBranchPrefix = "enrtree-branch:"
)

type entryRoot struct {
	eroot string
	lroot string
	sig   string
	seq   uint
}

func parseEntryRoot(s string) (*entryRoot, error) {
	if !strings.HasPrefix(s, entryRootPrefix) {
		return nil, fmt.Errorf("root entry does not have correct prefix '%s'", entryRootPrefix)
	}
	var eroot, lroot, sig string
	var seq uint

	if _, err := fmt.Sscanf(s, entryRootPrefix+" e=%s l=%s seq=%d sig=%s", &eroot, &lroot, &seq, &sig); err != nil {
		return nil, err
	}
	entry := &entryRoot{
		eroot: eroot,
		lroot: lroot,
		sig:   sig,
		seq:   seq,
	}
	return entry, nil
}

type entryBranch struct {
	hashes []string
}

func parseBranchRoot(s string) (*entryBranch, error) {
	if !strings.HasPrefix(s, entryBranchPrefix) {
		return nil, fmt.Errorf("root entry does not have correct prefix '%s'", entryRootPrefix)
	}
	s = s[len(entryBranchPrefix):]
	hashes := []string{}
	for _, hash := range strings.Split(s, ",") {
		hashes = append(hashes, hash)
	}
	entry := &entryBranch{
		hashes: hashes,
	}
	return entry, nil
}

func parseENR(s string) (*enr.Record, error) {
	return enr.Unmarshal(s)
}

func parseEntry(s string) (interface{}, error) {
	switch {
	case strings.HasPrefix(s, entryENRPrefix):
		return parseENR(s)
	case strings.HasPrefix(s, entryBranchPrefix):
		return parseBranchRoot(s)
	default:
		panic("BUG")
	}
}
