package vecfc

import (
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/kvdb"
)

func (vi *Index) getBytes(table kvdb.Store, id hash.Event) []byte {
	key := id.Bytes()
	b, err := table.Get(key)
	if err != nil {
		vi.ErrorHandler(err)
	}
	return b
}

func (vi *Index) setBytes(table kvdb.Store, id hash.Event, b []byte) {
	key := id.Bytes()
	err := table.Put(key, b)
	if err != nil {
		vi.ErrorHandler(err)
	}
}

// GetLowestAfter reads the vector from DB
func (vi *Index) GetLowestAfter(id hash.Event) *LowestAfterSeq {
	if bVal, okGet := vi.Cache.LowestAfterSeq.Get(id); okGet {
		return bVal.(*LowestAfterSeq)
	}

	b := LowestAfterSeq(vi.getBytes(vi.TableAllVecs.LowestAfterSeq, id))
	if b == nil {
		return nil
	}
	vi.Cache.LowestAfterSeq.Add(id, &b, uint(len(b)))
	return &b
}

// GetHighestBefore reads the vector from DB
func (vi *Index) GetHighestBefore(id hash.Event) *HighestBeforeSeq {
	if bVal, okGet := vi.Cache.HighestBeforeSeq.Get(id); okGet {
		return bVal.(*HighestBeforeSeq)
	}

	b := HighestBeforeSeq(vi.getBytes(vi.TableAllVecs.HighestBeforeSeq, id))
	if b == nil {
		return nil
	}
	vi.Cache.HighestBeforeSeq.Add(id, &b, uint(len(b)))
	return &b
}

// SetLowestAfter stores the vector into DB
func (vi *Index) SetLowestAfter(id hash.Event, seq *LowestAfterSeq) {
	vi.setBytes(vi.TableAllVecs.LowestAfterSeq, id, *seq)

	vi.Cache.LowestAfterSeq.Add(id, seq, uint(len(*seq)))
}

// SetHighestBefore stores the vectors into DB
func (vi *Index) SetHighestBefore(id hash.Event, seq *HighestBeforeSeq) {
	vi.setBytes(vi.TableAllVecs.HighestBeforeSeq, id, *seq)

	vi.Cache.HighestBeforeSeq.Add(id, seq, uint(len(*seq)))
}
