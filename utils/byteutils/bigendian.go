// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package byteutils

import "encoding/binary"

// Uint64ToBigEndian converts uint64 to big endian bytes.
func Uint64ToBigEndian(n uint64) []byte {
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, n)
	return res
}

// BigEndianToUint64 converts uint64 from big endian bytes.
func BigEndianToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Uint32ToBigEndian converts uint32 to big endian bytes.
func Uint32ToBigEndian(n uint32) []byte {
	res := make([]byte, 4)
	binary.BigEndian.PutUint32(res, n)
	return res
}

// BigEndianToUint32 converts uint32 from big endian bytes.
func BigEndianToUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

// Uint16ToBigEndian converts uint16 to big endian bytes.
func Uint16ToBigEndian(n uint16) []byte {
	res := make([]byte, 2)
	binary.BigEndian.PutUint16(res, n)
	return res
}

// BigEndianToUint16 converts uint16 from big endian bytes.
func BigEndianToUint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}
