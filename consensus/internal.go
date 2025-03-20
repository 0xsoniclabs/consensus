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

import "github.com/0xsoniclabs/consensus/utils/byteutils"

// ValidatorIndex represents a normalized value of ValidatorID for slice/array packing purposes
type ValidatorIndex uint32

// Bytes gets the byte representation of the index.
func (v ValidatorIndex) Bytes() []byte {
	return byteutils.Uint32ToBigEndian(uint32(v))
}

// BytesToValidator converts bytes to validator index.
func BytesToValidator(b []byte) ValidatorIndex {
	return ValidatorIndex(byteutils.BigEndianToUint32(b))
}
