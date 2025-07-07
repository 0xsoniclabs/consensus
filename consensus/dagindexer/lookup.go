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
	vi.cache.EventId.Add(newKey, id)
	vi.setLastUId(newKey)
	return newKey
}

func (vi *Index) getEventByUId(key consensus.Seq) consensus.EventHash {
	if value, ok := vi.cache.EventId.Get(key); ok {
		return value.(consensus.EventHash)
	}
	b, err := vi.table.EventUidTable.Get(key.Bytes())
	if err != nil {
		vi.crit(err)
	}
	eventHash := consensus.BytesToEvent(b)
	vi.cache.EventId.Add(key, eventHash)
	return eventHash
}

func (vi *Index) setLastUId(key consensus.Seq) {
	k := []byte("ll")
	err := vi.table.EventUidTable.Put(k, key.Bytes())
	if err != nil {
		vi.crit(err)
	}
	vi.cache.LastEventId = key
}

func (vi *Index) getLastUId() consensus.Seq {
	if vi.cache.LastEventId != NilLastEventUid {
		return vi.cache.LastEventId
	}
	k := []byte("ll")
	w, err := vi.table.EventUidTable.Get(k)
	if err != nil {
		vi.crit(err)
	}
	if w == nil {
		vi.cache.LastEventId = consensus.Seq(0)
		return consensus.Seq(0)
	}
	seq := consensus.BytesToSeq(w)
	vi.cache.LastEventId = seq
	return seq
}
