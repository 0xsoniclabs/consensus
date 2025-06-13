package dagindexer

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

func (vi *Index) addEventUId(id consensus.EventHash) consensus.Seq {
	lastKey := vi.getLastUId()
	newKey := lastKey + 1
	err := vi.table.EventUidTable.Put(newKey.Bytes(), id.Bytes())
	if err != nil {
		vi.crit(err)
	}
	vi.setLastUId(newKey)
	return newKey
}

func (vi *Index) getEventByUId(key consensus.Seq) consensus.EventHash {
	b, err := vi.table.EventUidTable.Get(key.Bytes())
	if err != nil {
		vi.crit(err)
	}
	return consensus.BytesToEvent(b)
}

func (vi *Index) setLastUId(key consensus.Seq) {
	k := []byte("ll")
	err := vi.table.EventUidTable.Put(k, key.Bytes())
	if err != nil {
		vi.crit(err)
	}
}

func (vi *Index) getLastUId() consensus.Seq {
	k := []byte("ll")
	w, err := vi.table.EventUidTable.Get(k)
	if err != nil {
		vi.crit(err)
	}
	if w == nil {
		return consensus.Seq(0)
	}
	return consensus.BytesToSeq(w)
}
