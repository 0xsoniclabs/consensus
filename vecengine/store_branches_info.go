package vecengine

import (
	"errors"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/kvdb"
)

func (vi *Engine) setRlp(table kvdb.Store, key []byte, val interface{}) {
	buf, err := rlp.EncodeToBytes(val)
	if err != nil {
		vi.ErrorHandler(err)
	}

	if err := table.Put(key, buf); err != nil {
		vi.ErrorHandler(err)
	}
}

func (vi *Engine) getRlp(table kvdb.Store, key []byte, to interface{}) interface{} {
	buf, err := table.Get(key)
	if err != nil {
		vi.ErrorHandler(err)
	}
	if buf == nil {
		return nil
	}

	err = rlp.DecodeBytes(buf, to)
	if err != nil {
		vi.ErrorHandler(err)
	}
	return to
}

func (vi *Engine) getBytes(table kvdb.Store, id hash.Event) []byte {
	key := id.Bytes()
	b, err := table.Get(key)
	if err != nil {
		vi.ErrorHandler(err)
	}
	return b
}

func (vi *Engine) setBytes(table kvdb.Store, id hash.Event, b []byte) {
	key := id.Bytes()
	err := table.Put(key, b)
	if err != nil {
		vi.ErrorHandler(err)
	}
}

func (vi *Engine) setBranchesInfo(info *BranchesInfo) {
	key := []byte("c")

	vi.setRlp(vi.Table.BranchesInfo, key, info)
}

func (vi *Engine) getBranchesInfo() *BranchesInfo {
	key := []byte("c")

	w, exists := vi.getRlp(vi.Table.BranchesInfo, key, &BranchesInfo{}).(*BranchesInfo)
	if !exists {
		return nil
	}

	return w
}

// SetEventBranchID stores the event's global branch ID
func (vi *Engine) SetEventBranchID(id hash.Event, branchID BranchID) {
	vi.setBytes(vi.Table.EventBranch, id, idx.Validator(branchID).Bytes()) // Cast done because idx.Validator has Bytes() method.
}

// GetEventBranchID reads the event's global branch ID
func (vi *Engine) GetEventBranchID(id hash.Event) BranchID {
	b := vi.getBytes(vi.Table.EventBranch, id)
	if b == nil {
		vi.ErrorHandler(errors.New("failed to read event's branch ID (inconsistent DB)"))
		return 0
	}
	branchID := BranchID(idx.BytesToValidator(b))
	return branchID
}
