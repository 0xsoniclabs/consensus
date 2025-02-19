package vecengine

import (
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

// ID of a branch
type BranchID = uint32

// BranchesInfo contains information about global branches of each validator
type BranchesInfo struct {
	BranchIDToHighestSeq         []idx.Event     // branchID -> highest e.Seq in the branch
	BranchIDToValidatorIndex     []idx.Validator // branchID -> validator idx
	ValidatorIndexToBranchIDList [][]BranchID    // validator idx -> list of branch IDs
}

// InitBranchesInfo loads BranchesInfo from store
func (vi *Engine) InitBranchesInfo() {
	if vi.BranchesInfo == nil {
		// if not cached
		vi.BranchesInfo = vi.getBranchesInfo()
		if vi.BranchesInfo == nil {
			// first run
			vi.BranchesInfo = newInitialBranchesInfo(vi.Validators)
		}
	}
}

func newInitialBranchesInfo(validators *pos.Validators) *BranchesInfo {
	branchIDCreators := validators.SortedIDs()
	branchIDToValidatorIndex := make([]idx.Validator, len(branchIDCreators))
	for i := range branchIDCreators {
		branchIDToValidatorIndex[i] = idx.Validator(i)
	}

	branchIDToHighestSeq := make([]idx.Event, len(branchIDToValidatorIndex))
	ValidatorIndexToBranchIDList := make([][]BranchID, validators.Len())
	for i := range ValidatorIndexToBranchIDList {
		ValidatorIndexToBranchIDList[i] = make([]BranchID, 1, validators.Len()/2+1)
		ValidatorIndexToBranchIDList[i][0] = BranchID(i)
	}
	return &BranchesInfo{
		BranchIDToHighestSeq:         branchIDToHighestSeq,
		BranchIDToValidatorIndex:     branchIDToValidatorIndex,
		ValidatorIndexToBranchIDList: ValidatorIndexToBranchIDList,
	}
}

func (vi *Engine) AtLeastOneFork() bool {
	return idx.Validator(len(vi.BranchesInfo.BranchIDToValidatorIndex)) > vi.Validators.Len()
}
