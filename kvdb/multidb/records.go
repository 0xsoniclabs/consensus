// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package multidb

import (
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/kvdb"
)

type TableRecord struct {
	Req   string
	Table string
}

func ReadTablesList(store kvdb.Store, tableRecordsKey []byte) (res []TableRecord, err error) {
	b, err := store.Get(tableRecordsKey)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return []TableRecord{}, nil
	}
	err = rlp.DecodeBytes(b, &res)
	return
}

func WriteTablesList(store kvdb.Store, tableRecordsKey []byte, records []TableRecord) error {
	b, err := rlp.EncodeToBytes(records)
	if err != nil {
		return err
	}
	return store.Put(tableRecordsKey, b)
}
