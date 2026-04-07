// Copyright (c) 2026 Sonic Operations Ltd

package consensus

// Block is a part of an ordered chain of batches of events.
type Block struct {
	Leader   EventHash
	Cheaters Cheaters
}
