package enr

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/umbracle/fastrlp"
)

type Record struct {
	seq       uint64
	signature []byte
	entries   entries
}

type Entry interface {
	MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value
	UnmarshalRLPWith(v *fastrlp.Value) error
}

func (r *Record) SetSeq(i uint64) {
	r.seq = i
}

func (r *Record) Seq() uint64 {
	return r.seq
}

func (r *Record) Signature() []byte {
	return r.signature
}

func (r *Record) Reset() {
	r.seq++
	r.signature = nil
	r.entries = entries{}
}

func (r *Record) Load(k string, v Entry) error {
	var found *entry
	for _, entry := range r.entries {
		if entry.k == k {
			found = &entry
			break
		}
	}
	if found == nil {
		return fmt.Errorf("key %s not found", k)
	}
	if err := v.UnmarshalRLPWith(found.v); err != nil {
		return err
	}
	return nil
}

func (r *Record) AddEntry(k string, v Entry) {
	ar := &fastrlp.Arena{}
	r.entries = append(r.entries, entry{
		k: k,
		v: v.MarshalRLPWith(ar),
	})
	r.entries.Sort()
}

func (r *Record) Marshal() string {
	return "enr:" + base64.RawURLEncoding.EncodeToString(r.MarshalRLP())
}

func (r *Record) MarshalRLP() []byte {
	ar := &fastrlp.Arena{}

	v := ar.NewArray()
	v.Set(ar.NewCopyBytes(r.signature))
	v.Set(ar.NewUint(r.seq))
	for _, entry := range r.entries {
		v.Set(ar.NewCopyBytes([]byte(entry.k)))
		v.Set(entry.v)
	}

	buf := v.MarshalTo(nil)
	return buf
}

func (r *Record) Unmarshal(s string) error {
	if !strings.HasPrefix(s, "enr:") {
		return fmt.Errorf("there is no enr prefix")
	}
	raw, err := base64.RawURLEncoding.DecodeString(s[4:])
	if err != nil {
		return err
	}
	if err := r.UnmarshalRLP(raw); err != nil {
		return err
	}
	return nil
}

func (r *Record) UnmarshalRLP(b []byte) error {
	p := &fastrlp.Parser{}
	v, err := p.Parse(b)
	if err != nil {
		return err
	}
	elems, err := v.GetElems()
	if err != nil {
		return err
	}
	if len(elems) < 2 {
		return fmt.Errorf("at least two items expected")
	}
	if len(elems)%2 != 0 {
		return fmt.Errorf("an even number of items expected")
	}

	if r.signature, err = elems[0].GetBytes(r.signature[:]); err != nil {
		return err
	}
	if r.seq, err = elems[1].GetUint64(); err != nil {
		return err
	}

	elems = elems[2:]
	r.entries = entries{}
	for i := 0; i < len(elems); i += 2 {
		entry := entry{}
		var dst []byte

		// name of the entry
		if dst, err = elems[i].GetBytes(nil); err != nil {
			return err
		}
		entry.k = string(dst)

		// value of the entry
		entry.v = elems[i+1]
		r.entries = append(r.entries, entry)
	}

	// check that the entry items are sorted
	for i := 0; i < len(r.entries)-1; i++ {
		a, b := r.entries[i], r.entries[i+1]

		if a.k == b.k {
			return fmt.Errorf("duplicated key %s", a.k)
		}
		if b.k < a.k {
			return fmt.Errorf("keys not sorted %s %s", a.k, b.k)
		}
	}
	return nil
}

func Unmarshal(b string) (*Record, error) {
	r := &Record{}
	if err := r.Unmarshal(b); err != nil {
		return nil, err
	}
	return r, nil
}

type entries []entry

func (e entries) Len() int {
	return len(e)
}

func (e entries) Less(i, j int) bool {
	return e[i].k < e[j].k
}

func (e entries) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e entries) Sort() {
	sort.Sort(e)
}

type entry struct {
	k string
	v *fastrlp.Value
}
