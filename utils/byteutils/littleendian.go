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

// Uint64ToLittleEndian converts uint64 to little endian byte.
func Uint64ToLittleEndian(n uint64) []byte {
	res := make([]byte, 8)
	binary.LittleEndian.PutUint64(res, n)
	return res
}

// LittleEndianToUint64 converts uint64 from little endian bytes.
func LittleEndianToUint64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}

// Uint32ToLittleEndian converts uint32 to little endian byte.
func Uint32ToLittleEndian(n uint32) []byte {
	res := make([]byte, 4)
	binary.LittleEndian.PutUint32(res, n)
	return res
}

// LittleEndianToUint32 converts uint32 from little endian bytes.
func LittleEndianToUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

// Uint16ToLittleEndian converts uint16 to little endian byte.
func Uint16ToLittleEndian(n uint16) []byte {
	res := make([]byte, 2)
	binary.LittleEndian.PutUint16(res, n)
	return res
}

// LittleEndianToUint16 converts uint16 from little endian bytes.
func LittleEndianToUint16(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}
