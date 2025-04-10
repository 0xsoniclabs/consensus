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
