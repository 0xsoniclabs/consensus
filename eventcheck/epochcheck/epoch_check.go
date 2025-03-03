// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: <TBD>
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package epochcheck

import (
	"errors"

	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

var (
	// ErrNotRelevant indicates the event's epoch isn't equal to current epoch.
	ErrNotRelevant = errors.New("event is too old or too new")
	// ErrAuth indicates that event's creator isn't authorized to create events in current epoch.
	ErrAuth = errors.New("event creator isn't a validator")
)

// Reader returns currents epoch and its validators group.
type Reader interface {
	GetEpochValidators() (*pos.Validators, idx.Epoch)
}

// Checker which require only current epoch info
type Checker struct {
	reader Reader
}

func New(reader Reader) *Checker {
	return &Checker{
		reader: reader,
	}
}

// Validate event
func (v *Checker) Validate(e dag.Event) error {
	// check epoch first, because validators group is returned only for the current epoch
	validators, epoch := v.reader.GetEpochValidators()
	if e.Epoch() != epoch {
		return ErrNotRelevant
	}
	if !validators.Exists(e.Creator()) {
		return ErrAuth
	}
	return nil
}
