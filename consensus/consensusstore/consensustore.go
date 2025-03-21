package consensusstore

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusengine/election"
)

type ConsensusStore interface {
	AddRoot(root consensus.Event)
	ApplyGenesis(g *Genesis) error
	Close() error
	DropEpochDB() error
	GetEpoch() consensus.Epoch
	GetEpochState() *EpochState
	GetEventConfirmedOn(e consensus.EventHash) consensus.Frame
	GetFrameRoots(frame consensus.Frame) []election.RootContext
	GetLastDecidedFrame() consensus.Frame
	GetLastDecidedState() *LastDecidedState
	GetValidators() *consensus.Validators
	OpenEpochDB(n consensus.Epoch) error
	SetEpochState(e *EpochState)
	SetEventConfirmedOn(e consensus.EventHash, on consensus.Frame)
	SetLastDecidedState(v *LastDecidedState)
	SwitchGenesis(g *Genesis) error
}
