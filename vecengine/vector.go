package vecengine

import (
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

type LowestAfterI interface {
	InitWithEvent(i BranchID, e dag.Event)
	Visit(i BranchID, e dag.Event) bool
}

type HighestBeforeI interface {
	InitWithEvent(i BranchID, e dag.Event)
	IsEmpty(i BranchID) bool
	IsForkDetected(i BranchID) bool
	Seq(i BranchID) idx.Event
	MinSeq(i BranchID) idx.Event
	SetForkDetected(i BranchID)
	CollectFrom(other HighestBeforeI, branchCount int)
	GatherFrom(to BranchID, other HighestBeforeI, from []BranchID)
}

type allVecs struct {
	after  LowestAfterI
	before HighestBeforeI
}
