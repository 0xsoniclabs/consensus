// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package ctype

import (
	"sync"
)

var (
	nodeNameDictMu  sync.RWMutex
	eventNameDictMu sync.RWMutex

	// nodeNameDict is an optional dictionary to make node address human readable in log.
	nodeNameDict = make(map[ValidatorID]string)

	// eventNameDict is an optional dictionary to make events human readable in log.
	eventNameDict = make(map[EventHash]string)
)

// SetNodeName sets an optional human readable alias of node address in log.
func SetNodeName(n ValidatorID, name string) {
	nodeNameDictMu.Lock()
	defer nodeNameDictMu.Unlock()

	nodeNameDict[n] = name
}

// SetEventName sets an optional human readable alias of event hash in log.
func SetEventName(e EventHash, name string) {
	eventNameDictMu.Lock()
	defer eventNameDictMu.Unlock()

	eventNameDict[e] = name
}

// GetNodeName gets an optional human readable alias of node address.
func GetNodeName(n ValidatorID) string {
	nodeNameDictMu.RLock()
	defer nodeNameDictMu.RUnlock()

	return nodeNameDict[n]
}

// GetEventName gets an optional human readable alias of event hash.
func GetEventName(e EventHash) string {
	eventNameDictMu.RLock()
	defer eventNameDictMu.RUnlock()

	return eventNameDict[e]
}
