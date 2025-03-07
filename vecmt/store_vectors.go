package vecmt

import (
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/vecengine"
)

func (vi *Index) getBytes(table kvdb.Store, id hash.Event) []byte {
	key := id.Bytes()
	b, err := table.Get(key)
	if err != nil {
		vi.crit(err)
	}
	return b
}

func (vi *Index) setBytes(table kvdb.Store, id hash.Event, b []byte) {
	key := id.Bytes()
	err := table.Put(key, b)
	if err != nil {
		vi.crit(err)
	}
}

// GetHighestBeforeTime reads the vector from DB
func (vi *Index) GetHighestBeforeTime(id hash.Event) *HighestBeforeTime {
	if bVal, okGet := vi.cache.HighestBeforeTime.Get(id); okGet {
		return bVal.(*HighestBeforeTime)
	}

	b := HighestBeforeTime(vi.getBytes(vi.table.HighestBeforeTime, id))
	if b == nil {
		return nil
	}
	vi.cache.HighestBeforeTime.Add(id, &b, uint(len(b)))
	return &b
}

// GetHighestBefore reads the vector from DB
func (vi *Index) GetHighestBefore(id hash.Event) *HighestBefore {
	return &HighestBefore{
		VSeq:  vi.Engine.GetHighestBefore(id),
		VTime: vi.GetHighestBeforeTime(id),
	}
}

func (vi *Index) GetLowestAfter(event hash.Event) vecengine.LowestAfterI {
	return vi.Engine.GetLowestAfter(event)
}

// SetHighestBeforeTime stores the vector into DB
func (vi *Index) SetHighestBeforeTime(id hash.Event, vec *HighestBeforeTime) {
	vi.setBytes(vi.table.HighestBeforeTime, id, *vec)
	vi.cache.HighestBeforeTime.Add(id, vec, uint(len(*vec)))
}

// SetHighestBefore stores the vectors into DB
func (vi *Index) SetHighestBefore(id hash.Event, vec *HighestBefore) {
	vi.Engine.SetHighestBefore(id, vec.VSeq)
	vi.SetHighestBeforeTime(id, vec.VTime)
}

func (vi *Index) SetLowestAfter(event hash.Event, i vecengine.LowestAfterI) {
	vi.Engine.SetLowestAfter(event, i.(*vecengine.LowestAfterSeq))
}

func (vi *Index) NewHighestBefore(size idx.Validator) vecengine.HighestBeforeI {
	return NewHighestBefore(size)
}

func (vi *Index) OnDropNotFlushed() {
	vi.Engine.OnDropNotFlushed()
	vi.onDropNotFlushed()
}
