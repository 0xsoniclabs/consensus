package dagindexer

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/kvdb"
)

func (vi *Index) getBytes(table kvdb.Store, id consensus.EventHash) []byte {
	key := id.Bytes()
	b, err := table.Get(key)
	if err != nil {
		vi.crit(err)
	}
	return b
}

func (vi *Index) setBytes(table kvdb.Store, id consensus.EventHash, b []byte) {
	key := id.Bytes()
	err := table.Put(key, b)
	if err != nil {
		vi.crit(err)
	}
}

// GetHighestBefore reads the vector from DB
func (vi *Index) GetHighestBefore(id consensus.EventHash) *HighestBefore {
	var vSeq *HighestBeforeSeq = nil
	if vSeqVal, ok := vi.cache.HighestBeforeSeq.Get(id); ok {
		vSeq = vSeqVal.(*HighestBeforeSeq) // Assertion needed because of raw bytes.
	} else {
		vSeqVal := HighestBeforeSeq(vi.getBytes(vi.table.HighestBeforeSeq, id))
		if vSeqVal != nil {
			vSeq = &vSeqVal
			vi.cache.HighestBeforeSeq.Add(id, vSeq, uint(len(*vSeq)))
		}
	}

	var vTime *HighestBeforeTime = nil
	if vTimeVal, ok := vi.cache.HighestBeforeTime.Get(id); ok {
		vTime = vTimeVal.(*HighestBeforeTime) // Assertion needed because of raw bytes.
	} else {
		vTimeVal := HighestBeforeTime(vi.getBytes(vi.table.HighestBeforeTime, id))
		if vTimeVal != nil {
			vTime = &vTimeVal
			vi.cache.HighestBeforeTime.Add(id, vTime, uint(len(*vTime)))
		}
	}

	if vSeq != nil && vTime != nil {
		return &HighestBefore{
			VSeq:  vSeq,
			VTime: vTime,
		}
	} else {
		return nil
	}
}

// GetLowestAfter reads the vector from DB
func (vi *Index) GetLowestAfter(id consensus.EventHash) *LowestAfter {
	if bVal, okGet := vi.cache.LowestAfterSeq.Get(id); okGet {
		return bVal.(*LowestAfter) // Cast needed because simplewlru uses raw interface{}.
	}

	b := LowestAfter(vi.getBytes(vi.table.LowestAfterSeq, id))
	if b == nil {
		return nil
	}
	vi.cache.LowestAfterSeq.Add(id, &b, uint(len(b)))
	return &b
}

// SetHighestBefore stores the vectors into DB
func (vi *Index) SetHighestBefore(id consensus.EventHash, vec *HighestBefore) {
	vi.setBytes(vi.table.HighestBeforeTime, id, *vec.VTime)
	vi.cache.HighestBeforeTime.Add(id, vec.VTime, uint(len(*vec.VTime)))
	vi.setBytes(vi.table.HighestBeforeSeq, id, *vec.VSeq)
	vi.cache.HighestBeforeSeq.Add(id, vec.VSeq, uint(len(*vec.VSeq)))
}

// SetLowestAfter stores the vector into DB
func (vi *Index) SetLowestAfter(id consensus.EventHash, seq *LowestAfterSeq) {
	vi.setBytes(vi.table.LowestAfterSeq, id, *seq)
	vi.cache.LowestAfterSeq.Add(id, seq, uint(len(*seq)))
}

func (vi *Index) OnDropNotFlushed() {
	vi.cache.HighestBeforeSeq.Purge()
	vi.cache.LowestAfterSeq.Purge()
	vi.cache.HighestBeforeTime.Purge()
}
