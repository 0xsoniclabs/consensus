package dagindexer

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/utils/byteutils"
)

type key uint64

func keyFromPair(branchID consensus.ValidatorIndex, seq consensus.Seq) key {
	return key(seq) | (key(branchID) << 32)
}

func (k key) Bytes() []byte {
	return byteutils.Uint64ToBigEndian(uint64(k))
}

func (vi *Index) addEventUId(k key, eventHash consensus.EventHash) consensus.Seq {
	err := vi.table.EventUidTable.Put(k.Bytes(), eventHash.Bytes())
	if err != nil {
		vi.crit(err)
	}
	vi.cache.EventId.Add(k, eventHash)
	return 0
}

func (vi *Index) getEventByUId(key key) consensus.EventHash {
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
