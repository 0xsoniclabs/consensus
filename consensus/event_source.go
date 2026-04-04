// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

// EventSource is a callback for getting events from an external storage.
type EventSource interface {
	HasEvent(EventHash) bool
	GetEvent(EventHash) Event
}
