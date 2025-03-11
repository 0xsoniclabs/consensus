// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package cachescale

import "github.com/0xsoniclabs/consensus/ctype"

type Func interface {
	I(int) int
	I32(int32) int32
	I64(int64) int64
	U(uint) uint
	U32(uint32) uint32
	U64(uint64) uint64
	F32(float32) float32
	F64(float64) float64
	Events(v ctype.Seq) ctype.Seq
	Blocks(v ctype.Block) ctype.Block
	Frames(v ctype.Frame) ctype.Frame
}
