// Copyright (c) 2026 Sonic Operations Ltd

package consensusengine

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

// EventSource is a callback for getting events from an external storage.
type EventSource interface {
	HasEvent(consensus.EventHash) bool
	GetEvent(consensus.EventHash) consensus.Event
}
