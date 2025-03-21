package consensusstore

import (
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
)

// LastDecidedState is for persistent storing.
type LastDecidedState struct {
	// fields can change only after a frame is decided
	LastDecidedFrame consensus.Frame
}

type EpochState struct {
	// stored values
	// these values change only after a change of epoch
	Epoch      consensus.Epoch
	Validators *consensus.Validators
}

func (es EpochState) String() string {
	return fmt.Sprintf("%d/%s", es.Epoch, es.Validators.String())
}
