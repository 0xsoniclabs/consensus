// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

import (
	"github.com/ethereum/go-ethereum/rlp"
)

// Cheaters is a slice type for storing cheaters list.
type Cheaters []ValidatorID

// Set returns map of cheaters
func (s Cheaters) Set() map[ValidatorID]struct{} {
	set := map[ValidatorID]struct{}{}
	for _, element := range s {
		set[element] = struct{}{}
	}
	return set
}

// Len returns the length of s.
func (s Cheaters) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s Cheaters) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// getRlp implements Rlpable and returns the i'th element of s in rlp.
func (s Cheaters) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}
