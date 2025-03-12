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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IntToBytes(t *testing.T) {
	assertar := assert.New(t)
	for _, n1 := range []uint64{
		0,
		9,
		0xF000000000000000,
		0x000000000000000F,
		0xFFFFFFFFFFFFFFFF,
		47528346792,
	} {
		{
			b := Uint64ToBigEndian(n1)
			assertar.Equal(8, len(b))
			n2 := BigEndianToUint64(b)
			assertar.Equal(n1, n2)
		}
		{
			b := Uint64ToLittleEndian(n1)
			assertar.Equal(8, len(b))
			n2 := LittleEndianToUint64(b)
			assertar.Equal(n1, n2)
		}
	}
	for _, n1 := range []uint32{
		0,
		9,
		0xFFFFFFFF,
		475283467,
	} {
		{
			b := Uint32ToBigEndian(n1)
			assertar.Equal(4, len(b))
			n2 := BigEndianToUint32(b)
			assertar.Equal(n1, n2)
		}
		{
			b := Uint32ToLittleEndian(n1)
			assertar.Equal(4, len(b))
			n2 := LittleEndianToUint32(b)
			assertar.Equal(n1, n2)
		}
	}
	for _, n1 := range []uint16{
		0,
		9,
		0xFFFF,
		47528,
	} {
		{
			b := Uint16ToBigEndian(n1)
			assertar.Equal(2, len(b))
			n2 := BigEndianToUint16(b)
			assertar.Equal(n1, n2)
		}
		{
			b := Uint16ToLittleEndian(n1)
			assertar.Equal(2, len(b))
			n2 := LittleEndianToUint16(b)
			assertar.Equal(n1, n2)
		}
	}
}
