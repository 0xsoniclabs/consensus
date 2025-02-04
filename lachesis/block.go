package lachesis

import (
	"github.com/0xsoniclabs/consensus/hash"
)

// Block is a part of an ordered chain of batches of events.
type Block struct {
	Atropos  hash.Event
	Cheaters Cheaters
}
