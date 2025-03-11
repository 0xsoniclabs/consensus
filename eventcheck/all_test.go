// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package eventcheck

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/ctype"
	"github.com/0xsoniclabs/consensus/eventcheck/basiccheck"
	"github.com/0xsoniclabs/consensus/eventcheck/epochcheck"
	"github.com/0xsoniclabs/consensus/eventcheck/parentscheck"
)

type testReader struct{}

func (tr *testReader) GetEpochValidators() (*ctype.Validators, ctype.Epoch) {
	vb := ctype.NewBuilder()
	vb.Set(1, 1)
	return vb.Build(), 1
}

func TestBasicEventValidation(t *testing.T) {
	var tests = []struct {
		e       ctype.Event
		wantErr error
	}{
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(), nil},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(0)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(), basiccheck.ErrNotInited},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(), basiccheck.ErrNoParents},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(math.MaxInt32 - 1)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(), basiccheck.ErrHugeValue},
	}

	for _, tt := range tests {
		basicCheck := basiccheck.New()
		assert.Equal(t, tt.wantErr, basicCheck.Validate(tt.e))
	}
}

func TestEpochEventValidation(t *testing.T) {
	var tests = []struct {
		e       ctype.Event
		wantErr error
	}{
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetEpoch(1)
			e.SetCreator(1)
			return e
		}(), nil},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetEpoch(2)
			e.SetCreator(1)
			return e
		}(), epochcheck.ErrNotRelevant},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetEpoch(1)
			e.SetCreator(2)
			return e
		}(), epochcheck.ErrAuth},
	}

	for _, tt := range tests {
		tr := new(testReader)
		epochCheck := epochcheck.New(tr)
		assert.Equal(t, tt.wantErr, epochCheck.Validate(tt.e))
	}
}

func TestParentsEventValidation(t *testing.T) {
	var tests = []struct {
		e         ctype.Event
		pe        ctype.Events
		wantErr   error
		wantPanic bool
	}{
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			e.SetCreator(1)
			selfParent := &ctype.TestEvent{}
			selfParent.SetLamport(1)
			selfParent.SetID([24]byte{1})
			e.SetParents(ctype.EventHashes{selfParent.ID()})
			return e
		}(),
			func() ctype.Events {
				e := &ctype.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				e.SetCreator(1)
				e.SetID([24]byte{1})
				return ctype.Events{e}
			}(),
			nil, false},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			e.SetCreator(1)
			selfParent := &ctype.TestEvent{}
			selfParent.SetLamport(1)
			selfParent.SetID([24]byte{2})
			e.SetParents(ctype.EventHashes{selfParent.ID()})
			return e
		}(),
			func() ctype.Events {
				e := &ctype.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				e.SetCreator(1)
				e.SetID([24]byte{1})
				return ctype.Events{e}
			}(),
			parentscheck.ErrWrongSelfParent, false},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(1)
			e.SetParents(ctype.EventHashes{e.ID()})
			return e
		}(),
			func() ctype.Events {
				e := &ctype.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				return ctype.Events{e}
			}(),
			parentscheck.ErrWrongLamport, false},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(2)
			e.SetParents(ctype.EventHashes{e.ID()})
			return e
		}(),
			func() ctype.Events {
				e := &ctype.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				return ctype.Events{e}
			}(),
			parentscheck.ErrWrongSelfParent, false},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			selfParent := &ctype.TestEvent{}
			selfParent.SetLamport(1)
			selfParent.SetID([24]byte{1})
			e.SetParents(ctype.EventHashes{selfParent.ID()})
			return e
		}(),
			func() ctype.Events {
				e := &ctype.TestEvent{}
				e.SetSeq(2)
				e.SetLamport(1)
				e.SetID([24]byte{1})
				return ctype.Events{e}
			}(),
			parentscheck.ErrWrongSeq, false},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(1)
			return e
		}(),
			nil,
			parentscheck.ErrWrongSeq, false},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(1)
			e.SetParents(ctype.EventHashes{e.ID()})
			return e
		}(),
			nil,
			nil, true},
	}

	for _, tt := range tests {
		parentsCheck := parentscheck.New()
		if tt.wantPanic {
			assert.Panics(t, func() {
				err := parentsCheck.Validate(tt.e, tt.pe)
				if err != nil {
					return
				}
			})
		} else {
			assert.Equal(t, tt.wantErr, parentsCheck.Validate(tt.e, tt.pe))
		}
	}
}

func TestAllEventValidation(t *testing.T) {
	var tests = []struct {
		e       ctype.Event
		pe      ctype.Events
		wantErr error
	}{
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			e.SetParents(ctype.EventHashes{e.ID()})
			return e
		}(),
			nil,
			basiccheck.ErrNotInited},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			return e
		}(),
			nil,
			epochcheck.ErrAuth},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(2)
			e.SetLamport(2)
			e.SetCreator(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			e.SetParents(ctype.EventHashes{e.ID()})
			return e
		}(),
			func() ctype.Events {
				e := &ctype.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				return ctype.Events{e}
			}(),
			parentscheck.ErrWrongSelfParent},
		{func() ctype.Event {
			e := &ctype.TestEvent{}
			e.SetSeq(1)
			e.SetLamport(2)
			e.SetCreator(1)
			e.SetEpoch(1)
			e.SetFrame(1)
			e.SetParents(ctype.EventHashes{e.ID()})
			return e
		}(),
			func() ctype.Events {
				e := &ctype.TestEvent{}
				e.SetSeq(1)
				e.SetLamport(1)
				return ctype.Events{e}
			}(),
			nil},
	}

	tr := new(testReader)

	checkers := Checkers{
		Basiccheck:   basiccheck.New(),
		Epochcheck:   epochcheck.New(tr),
		Parentscheck: parentscheck.New(),
	}

	for _, tt := range tests {
		assert.Equal(t, tt.wantErr, checkers.Validate(tt.e, tt.pe))
	}
}
