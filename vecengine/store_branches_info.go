// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vecengine

import (
	"errors"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/kvdb"
)

func (vi *Engine) setRlp(table kvdb.Store, key []byte, val interface{}) {
	buf, err := rlp.EncodeToBytes(val)
	if err != nil {
		vi.crit(err)
	}

	if err := table.Put(key, buf); err != nil {
		vi.crit(err)
	}
}

func (vi *Engine) getRlp(table kvdb.Store, key []byte, to interface{}) interface{} {
	buf, err := table.Get(key)
	if err != nil {
		vi.crit(err)
	}
	if buf == nil {
		return nil
	}

	err = rlp.DecodeBytes(buf, to)
	if err != nil {
		vi.crit(err)
	}
	return to
}

func (vi *Engine) setBranchesInfo(info *BranchesInfo) {
	key := []byte("c")

	vi.setRlp(vi.table.BranchesInfo, key, info)
}

func (vi *Engine) getBranchesInfo() *BranchesInfo {
	key := []byte("c")

	w, exists := vi.getRlp(vi.table.BranchesInfo, key, &BranchesInfo{}).(*BranchesInfo)
	if !exists {
		return nil
	}

	return w
}

// SetEventBranchID stores the event's global branch ID
func (vi *Engine) SetEventBranchID(id consensus.EventHash, branchID consensus.ValidatorIndex) {
	vi.setBytes(vi.table.EventBranch, id, branchID.Bytes())
}

// GetEventBranchID reads the event's global branch ID
func (vi *Engine) GetEventBranchID(id consensus.EventHash) consensus.ValidatorIndex {
	b := vi.getBytes(vi.table.EventBranch, id)
	if b == nil {
		vi.crit(errors.New("failed to read event's branch ID (inconsistent DB)"))
		return 0
	}
	branchID := consensus.BytesToValidator(b)
	return branchID
}
