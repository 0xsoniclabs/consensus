// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagindexer

import (
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagbranch"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagstore"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagvec"
)

// Compile-time check that dagstore.Store satisfies dagbranch.BranchStore.
var _ dagbranch.BranchStore = (*dagstore.Store)(nil)

// Type aliases re-exported from subpackages for external consumers.
type (
	Timestamp        = dagvec.Timestamp
	HighestBeforeSeq = dagvec.HighestBeforeSeq
	BranchSeq        = dagvec.BranchSeq
)
