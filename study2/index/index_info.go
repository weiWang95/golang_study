package index

import (
	"encoding/binary"
	"fmt"
)

type IndexInfo []byte

const IndexInfoLen = 128

func NewIndexInfo() IndexInfo {
	return make([]byte, IndexInfoLen)
}

func (i IndexInfo) Total() uint64 {
	return binary.BigEndian.Uint64(i[8:16])
}

func (i IndexInfo) SetTotal(total uint64) {
	binary.BigEndian.PutUint64(i[8:16], total)
}

func (i IndexInfo) IncTotal(inc uint64) {
	i.SetTotal(i.Total() + inc)
}

func (i IndexInfo) String() string {
	return fmt.Sprintf("t:%d", i.Total())
}
