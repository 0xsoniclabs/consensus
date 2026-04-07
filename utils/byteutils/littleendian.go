// Copyright (c) 2026 Sonic Operations Ltd

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
