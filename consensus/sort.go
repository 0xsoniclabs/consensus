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

type (
	validator struct {
		ID     ValidatorID
		Weight Weight
	}

	validators []validator
)

func (vv validators) Less(i, j int) bool {
	if vv[i].Weight != vv[j].Weight {
		return vv[i].Weight > vv[j].Weight
	}

	return vv[i].ID < vv[j].ID
}

func (vv validators) Len() int {
	return len(vv)
}

func (vv validators) Swap(i, j int) {
	vv[i], vv[j] = vv[j], vv[i]
}
