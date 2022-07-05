package enr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestENR(t *testing.T) {
	enrStr := "enr:-IS4QHCYrYZbAKWCBRlAy5zzaDZXJBGkcnh4MHcBFZntXNFrdvJjX04jRzjzCBOonrkTfj499SZuOh8R33Ls8RRcy5wBgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQPKY0yuDUmstAHYpMa2_oxVtw0RW_QAdpzBQA8yWM0xOIN1ZHCCdl8"
	record, err := Unmarshal(enrStr)
	assert.NoError(t, err)

	var ip IPv4
	assert.NoError(t, record.Load("ip", &ip))
	assert.Equal(t, ip, IPv4([]byte{127, 0, 0, 1}))

	var udp Uint16
	assert.NoError(t, record.Load("udp", &udp))
	assert.Equal(t, udp, Uint16(30303))

	found := record.Marshal()
	assert.Equal(t, enrStr, found)
}
