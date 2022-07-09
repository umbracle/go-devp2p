package dnsdisc

import (
	"crypto/ecdsa"
	base32Go "encoding/base32"
	"fmt"
	"strings"

	"github.com/umbracle/go-devp2p/crypto"
	"github.com/umbracle/go-devp2p/enr"
)

type Entry interface {
	isEntry()
	String() string
}

type entryImpl struct {
}

func (e *entryImpl) isEntry() {
}

var (
	base32 = base32Go.StdEncoding.WithPadding(base32Go.NoPadding)
)

// entries prefixes
var (
	entryENRPrefix    = "enr:"
	entryRootPrefix   = "enrtree-root:v1"
	entryBranchPrefix = "enrtree-branch:"
	entryLinkPrefix   = "enrtree://"
)

type entryRoot struct {
	entryImpl

	eroot string
	lroot string
	sig   string
	seq   uint
}

func (e *entryRoot) String() string {
	return fmt.Sprintf(entryRootPrefix+" e=%s l=%s seq=%d sig=%s", e.eroot, e.lroot, e.seq, e.sig)
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

	if err := validateHash(eroot); err != nil {
		return nil, err
	}
	if err := validateHash(lroot); err != nil {
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
	entryImpl

	hashes []string
}

func (e *entryBranch) String() string {
	return fmt.Sprintf(entryBranchPrefix + strings.Join(e.hashes, ","))
}

func parseBranchRoot(s string) (*entryBranch, error) {
	var err error
	if s, err = checkAndTrimPrefix(s, entryBranchPrefix); err != nil {
		return nil, err
	}
	hashes := []string{}
	for _, hash := range strings.Split(s, ",") {
		if err := validateHash(hash); err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}
	entry := &entryBranch{
		hashes: hashes,
	}
	return entry, nil
}

type enrEntry struct {
	entryImpl

	record *enr.Record
}

func (e *enrEntry) String() string {
	return e.record.Marshal()
}

func parseENR(s string) (*enrEntry, error) {
	record, err := enr.Unmarshal(s)
	if err != nil {
		return nil, err
	}
	return &enrEntry{record: record}, nil
}

type entryLink struct {
	entryImpl

	pubKey *ecdsa.PublicKey
	domain string
}

func (e *entryLink) String() string {
	pubKeyBase32 := base32.EncodeToString(crypto.CompressPubKey(e.pubKey))
	return entryLinkPrefix + pubKeyBase32 + "@" + e.domain
}

func parseEntryLink(s string) (*entryLink, error) {
	var err error
	if s, err = checkAndTrimPrefix(s, entryLinkPrefix); err != nil {
		return nil, err
	}

	parts := strings.Split(s, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("incorrect format")
	}

	pubKeyBase64, domain := parts[0], parts[1]

	pubKey, err := base32.DecodeString(pubKeyBase64)
	if err != nil {
		return nil, err
	}
	key, err := crypto.ParseCompressedPubKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to process pub key: %v", err)
	}
	return &entryLink{pubKey: key, domain: domain}, nil
}

func parseEntry(s string) (Entry, error) {
	switch {
	case strings.HasPrefix(s, entryENRPrefix):
		return parseENR(s)
	case strings.HasPrefix(s, entryBranchPrefix):
		return parseBranchRoot(s)
	case strings.HasPrefix(s, entryLinkPrefix):
		return parseEntryLink(s)
	case strings.HasPrefix(s, entryRootPrefix):
		return parseEntryRoot(s)
	default:
		return nil, fmt.Errorf("BUG: entry type not expected %s", s)
	}
}

func checkAndTrimPrefix(s string, prefix string) (string, error) {
	if !strings.HasPrefix(s, prefix) {
		return "", fmt.Errorf("entry does not have correct prefix '%s'", prefix)
	}
	s = s[len(prefix):]
	return s, nil
}

func validateHash(s string) error {
	buf, err := base32.DecodeString(s)
	if err != nil {
		return err
	}
	if size := len(buf); size < 12 || size > 32 {
		return fmt.Errorf("incorrect length, expected [12,32] but found %d", len(buf))
	}
	return nil
}
