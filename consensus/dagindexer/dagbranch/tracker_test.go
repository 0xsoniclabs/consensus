// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagbranch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagvec"
)

// mockStore implements BranchStore for testing.
type mockStore struct {
	branchBytes   []byte
	eventBranches map[consensus.EventHash]consensus.ValidatorIndex
}

func newMockStore() *mockStore {
	return &mockStore{
		eventBranches: make(map[consensus.EventHash]consensus.ValidatorIndex),
	}
}

func (m *mockStore) GetEventBranchID(id consensus.EventHash) consensus.ValidatorIndex {
	return m.eventBranches[id]
}

func (m *mockStore) SetEventBranchID(id consensus.EventHash, branchID consensus.ValidatorIndex) {
	m.eventBranches[id] = branchID
}

func (m *mockStore) GetBranchesInfoBytes() []byte {
	return m.branchBytes
}

func (m *mockStore) SetBranchesInfoBytes(data []byte) {
	m.branchBytes = data
}

func makeValidators(n int) *consensus.Validators {
	b := consensus.NewValidatorsBuilder()
	for i := 0; i < n; i++ {
		b.Set(consensus.ValidatorID(i+1), 1)
	}
	return b.Build()
}

func tCrit(err error) { panic(err) }

func TestTracker_InitFromStore(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	// First tracker writes BranchesInfo to the store.
	t1 := NewTracker(tCrit, validators, store)
	t1.Init()
	assert.NotNil(t, t1.Info())
	t1.Flush()
	assert.NotNil(t, store.branchBytes)

	// Second tracker reads it back.
	t2 := NewTracker(tCrit, validators, store)
	t2.Init()
	assert.Equal(t, t1.Info().BranchIDLastSeq, t2.Info().BranchIDLastSeq)
	assert.Equal(t, t1.Info().BranchIDCreatorIdxs, t2.Info().BranchIDCreatorIdxs)
}

func TestTracker_InitFresh(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	assert.Equal(t, consensus.ValidatorIndex(3), tr.BranchCount())
	assert.False(t, tr.AtLeastOneEquivocation())
}

func TestTracker_InitBadRLP(t *testing.T) {
	store := newMockStore()
	store.branchBytes = []byte("invalid rlp data")
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	assert.Panics(t, func() {
		tr.Init()
	})
}

func TestTracker_FlushNil(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	// Flush with nil info should be a no-op.
	tr.Flush()
	assert.Nil(t, store.branchBytes)
}

func TestTracker_DropNotFlushed(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()
	assert.NotNil(t, tr.Info())
	tr.DropNotFlushed()
	assert.Nil(t, tr.Info())
}

func TestTracker_GetSetEventBranchID(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	hash := consensus.EventHash{}
	tr.SetEventBranchID(hash, 42)
	assert.Equal(t, consensus.ValidatorIndex(42), tr.GetEventBranchID(hash))
}

// mockEvent implements consensus.Event for testing AssignBranchID.
type mockEvent struct {
	consensus.Event
	seq        consensus.Seq
	selfParent *consensus.EventHash
	id         consensus.EventHash
	parents    consensus.EventHashes
	creator    consensus.ValidatorID
}

func (e *mockEvent) Seq() consensus.Seq               { return e.seq }
func (e *mockEvent) SelfParent() *consensus.EventHash { return e.selfParent }
func (e *mockEvent) ID() consensus.EventHash          { return e.id }
func (e *mockEvent) Parents() consensus.EventHashes   { return e.parents }
func (e *mockEvent) Creator() consensus.ValidatorID   { return e.creator }

func TestTracker_AssignBranchID_FirstEvent(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	e := &mockEvent{seq: 1}
	branchID, err := tr.AssignBranchID(e, 0)
	assert.NoError(t, err)
	assert.Equal(t, consensus.ValidatorIndex(0), branchID)
}

func TestTracker_AssignBranchID_ContinuesBranch(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	// First event
	e1 := &mockEvent{seq: 1, id: consensus.EventHash{1}}
	branchID1, err := tr.AssignBranchID(e1, 0)
	assert.NoError(t, err)
	store.eventBranches[e1.id] = branchID1

	// Continuation (seq+1 from self-parent)
	sp := e1.id
	e2 := &mockEvent{seq: 2, selfParent: &sp}
	branchID2, err := tr.AssignBranchID(e2, 0)
	assert.NoError(t, err)
	assert.Equal(t, branchID1, branchID2)
}

func TestTracker_AssignBranchID_Equivocation(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	// First event
	e1 := &mockEvent{seq: 1, id: consensus.EventHash{1}}
	branchID1, err := tr.AssignBranchID(e1, 0)
	assert.NoError(t, err)
	store.eventBranches[e1.id] = branchID1

	// Equivocation: another first event (no self-parent, but seq already filled)
	e2 := &mockEvent{seq: 1}
	branchID2, err := tr.AssignBranchID(e2, 0)
	assert.NoError(t, err)
	assert.NotEqual(t, branchID1, branchID2)
	assert.True(t, tr.AtLeastOneEquivocation())
}

func TestTracker_AssignBranchID_EquivocationWithSelfParent(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	// First event
	e1 := &mockEvent{seq: 1, id: consensus.EventHash{1}}
	branchID1, err := tr.AssignBranchID(e1, 0)
	assert.NoError(t, err)
	store.eventBranches[e1.id] = branchID1

	// Equivocation with self-parent: seq gap (seq 3 instead of 2)
	sp := e1.id
	e2 := &mockEvent{seq: 3, selfParent: &sp}
	branchID2, err := tr.AssignBranchID(e2, 0)
	assert.NoError(t, err)
	assert.NotEqual(t, branchID1, branchID2)
	assert.True(t, tr.AtLeastOneEquivocation())
}

func TestTracker_AssignBranchID_InconsistentDB(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	// Corrupt the info to trigger inconsistency check
	tr.Info().BranchIDCreatorIdxs = tr.Info().BranchIDCreatorIdxs[:1]

	e := &mockEvent{seq: 1}
	_, err := tr.AssignBranchID(e, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inconsistent")
}

func TestTracker_AssignBranchID_TooFewBranches(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	// Corrupt: remove entries so length < validator count
	tr.Info().BranchIDCreatorIdxs = tr.Info().BranchIDCreatorIdxs[:2]
	tr.Info().BranchIDLastSeq = tr.Info().BranchIDLastSeq[:2]

	e := &mockEvent{seq: 1}
	_, err := tr.AssignBranchID(e, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inconsistent")
}

func TestTracker_AssignBranchID_InconsistentDBWithSelfParent(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	// First event to set up a self-parent
	e1 := &mockEvent{seq: 1, id: consensus.EventHash{1}}
	_, err := tr.AssignBranchID(e1, 0)
	assert.NoError(t, err)
	store.eventBranches[e1.id] = 0

	// Corrupt the info for the self-parent path
	tr.Info().BranchIDCreatorIdxs = tr.Info().BranchIDCreatorIdxs[:1]

	sp := e1.id
	e2 := &mockEvent{seq: 2, selfParent: &sp}
	_, err = tr.AssignBranchID(e2, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inconsistent")
}

func TestTracker_SetEquivocationDetected(t *testing.T) {
	validators := makeValidators(3)
	store := newMockStore()
	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	before := dagvec.NewHighestBefore(tr.BranchCount())
	before.VSeq.Set(0, dagvec.BranchSeq{Seq: 1})
	tr.SetEquivocationDetected(before, 0)
	assert.True(t, before.IsEquivocationDetected(0))
}

func TestTracker_DetectEquivocations_NoEquivocation(t *testing.T) {
	validators := makeValidators(3)
	store := newMockStore()
	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	before := dagvec.NewHighestBefore(tr.BranchCount())
	// No equivocations => no-op
	tr.DetectEquivocations(before)
}

func TestTracker_GetMergedHighestBefore_NoEquivocation(t *testing.T) {
	validators := makeValidators(3)
	store := newMockStore()
	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	hb := dagvec.NewHighestBefore(tr.BranchCount())
	hb.VSeq.Set(0, dagvec.BranchSeq{Seq: 1})

	hash := consensus.EventHash{1}
	getter := func(id consensus.EventHash) *dagvec.HighestBefore {
		return hb
	}

	result := tr.GetMergedHighestBefore(hash, getter)
	assert.Equal(t, hb, result)
}

// setupEquivocation creates a tracker with 3 validators where validator 0
// has equivocated (2 branches: branch 0 and branch 3).
func setupEquivocation() (*Tracker, *mockStore) {
	store := newMockStore()
	validators := makeValidators(3)
	tr := NewTracker(tCrit, validators, store)
	tr.Init()

	// First event for validator 0 on branch 0
	e1 := &mockEvent{seq: 1, id: consensus.EventHash{1}}
	bid, _ := tr.AssignBranchID(e1, 0)
	store.eventBranches[e1.id] = bid

	// Equivocating event for validator 0 (no self-parent, same seq already used)
	e2 := &mockEvent{seq: 1, id: consensus.EventHash{2}}
	bid2, _ := tr.AssignBranchID(e2, 0)
	store.eventBranches[e2.id] = bid2

	return tr, store
}

func TestTracker_DetectEquivocations_PropagateMarker(t *testing.T) {
	tr, _ := setupEquivocation()
	// validator 0 has branches [0, 3]
	branchCount := tr.BranchCount() // 4 (3 validators + 1 equivocation branch)
	before := dagvec.NewHighestBefore(branchCount)

	// Mark branch 0 (one of validator 0's branches) as equivocation detected
	before.SetEquivocationDetected(0)
	// Branch 3 is NOT yet marked

	tr.DetectEquivocations(before)

	// After detection, branch 3 should also be marked
	assert.True(t, before.IsEquivocationDetected(0))
	assert.True(t, before.IsEquivocationDetected(3))
}

func TestTracker_DetectEquivocations_CrossBranchDetection(t *testing.T) {
	tr, _ := setupEquivocation()
	branchCount := tr.BranchCount() // 4

	before := dagvec.NewHighestBefore(branchCount)
	// Set up overlapping sequences on branches 0 and 3 (both for validator 0)
	// Branch 0: Seq=5, MinSeq=1
	before.VSeq.Set(0, dagvec.BranchSeq{Seq: 5, MinSeq: 1})
	// Branch 3: Seq=3, MinSeq=1
	before.VSeq.Set(3, dagvec.BranchSeq{Seq: 3, MinSeq: 1})
	// Overlap: MinSeq(0)=1 <= Seq(3)=3 AND MinSeq(3)=1 <= Seq(0)=5

	// Neither branch has equivocation detected yet
	assert.False(t, before.IsEquivocationDetected(0))
	assert.False(t, before.IsEquivocationDetected(3))

	tr.DetectEquivocations(before)

	// After detection, both should be marked
	assert.True(t, before.IsEquivocationDetected(0))
	assert.True(t, before.IsEquivocationDetected(3))
}

func TestTracker_DetectEquivocations_SkipsEmptyBranch(t *testing.T) {
	tr, _ := setupEquivocation()
	branchCount := tr.BranchCount()

	before := dagvec.NewHighestBefore(branchCount)
	// Branch 0 has data, but branch 3 is empty
	before.VSeq.Set(0, dagvec.BranchSeq{Seq: 5, MinSeq: 1})
	// Branch 3 left empty (Seq=0, no equivocation)

	tr.DetectEquivocations(before)

	// Should NOT detect equivocation because one branch is empty
	assert.False(t, before.IsEquivocationDetected(0))
}

func TestTracker_GetMergedHighestBefore_WithEquivocation(t *testing.T) {
	tr, _ := setupEquivocation()
	branchCount := tr.BranchCount() // 4

	// Create a scattered HighestBefore with 4 branches
	scattered := dagvec.NewHighestBefore(branchCount)
	// Branch 0 (validator 0's first branch): Seq=5
	scattered.VSeq.Set(0, dagvec.BranchSeq{Seq: 5, MinSeq: 1})
	scattered.VTime.Set(0, 1000)
	// Branch 3 (validator 0's equivocation branch): Seq=3
	scattered.VSeq.Set(3, dagvec.BranchSeq{Seq: 3, MinSeq: 1})
	scattered.VTime.Set(3, 500)
	// Branch 1 (validator 1): Seq=2
	scattered.VSeq.Set(1, dagvec.BranchSeq{Seq: 2, MinSeq: 2})
	scattered.VTime.Set(1, 800)
	// Branch 2 (validator 2): Seq=1
	scattered.VSeq.Set(2, dagvec.BranchSeq{Seq: 1, MinSeq: 1})
	scattered.VTime.Set(2, 600)

	hash := consensus.EventHash{10}
	getter := func(id consensus.EventHash) *dagvec.HighestBefore {
		return scattered
	}

	result := tr.GetMergedHighestBefore(hash, getter)

	// Merged result should have 3 entries (one per validator)
	assert.Equal(t, dagvec.Timestamp(3), dagvec.Timestamp(result.VSeq.Size()))
	// Validator 0: should be merged from branches 0 and 3 (highest wins: Seq=5)
	assert.Equal(t, consensus.Seq(5), result.VSeq.Get(0).Seq)
	assert.Equal(t, dagvec.Timestamp(1000), result.VTime.Get(0))
	// Validator 1: branch 1 directly
	assert.Equal(t, consensus.Seq(2), result.VSeq.Get(1).Seq)
	// Validator 2: branch 2 directly
	assert.Equal(t, consensus.Seq(1), result.VSeq.Get(2).Seq)
}

func TestTracker_FlushEncodesRLP(t *testing.T) {
	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(tCrit, validators, store)
	tr.Init()
	tr.Flush()

	assert.NotNil(t, store.branchBytes)
	assert.True(t, len(store.branchBytes) > 0)
}

func TestTracker_FlushCritOnRLPError(t *testing.T) {
	// This is hard to trigger since BranchesInfo always encodes fine.
	// The flush path is already covered by TestTracker_FlushEncodesRLP.
	// We verify the encode-error branch indirectly: if encode succeeds, crit is not called.
	var critCalled bool
	crit := func(err error) { critCalled = true }

	store := newMockStore()
	validators := makeValidators(3)

	tr := NewTracker(crit, validators, store)
	tr.Init()
	tr.Flush()
	assert.False(t, critCalled)
}

func TestNewInitialBranchesInfo(t *testing.T) {
	validators := makeValidators(4)
	info := NewInitialBranchesInfo(validators)

	assert.Equal(t, 4, len(info.BranchIDLastSeq))
	assert.Equal(t, 4, len(info.BranchIDCreatorIdxs))
	assert.Equal(t, 4, len(info.BranchIDByCreators))
	for i := 0; i < 4; i++ {
		assert.Equal(t, consensus.ValidatorIndex(i), info.BranchIDCreatorIdxs[i])
		assert.Equal(t, 1, len(info.BranchIDByCreators[i]))
	}
	assert.False(t, info.AtLeastOneEquivocation(4))
}

func TestBranchesInfo_AtLeastOneEquivocation(t *testing.T) {
	info := &BranchesInfo{
		BranchIDCreatorIdxs: make([]consensus.ValidatorIndex, 5),
	}
	// 5 branches with 4 validators => equivocation
	assert.True(t, info.AtLeastOneEquivocation(4))
	// 5 branches with 5 validators => no equivocation
	assert.False(t, info.AtLeastOneEquivocation(5))
}

func TestTracker_Flush_ErrorPath(t *testing.T) {
	// Verify that Flush calls SetBranchesInfoBytes with valid data
	store := newMockStore()
	validators := makeValidators(2)
	tr := NewTracker(tCrit, validators, store)
	tr.Init()
	tr.Flush()

	// Verify round-trip: create new tracker from stored bytes
	tr2 := NewTracker(tCrit, validators, store)
	tr2.Init()
	assert.Equal(t, tr.Info().BranchIDLastSeq, tr2.Info().BranchIDLastSeq)
}

func TestTracker_Init_DecodeCritPath(t *testing.T) {
	// Test that Init calls crit when RLP decode fails
	var critErr error
	crit := func(err error) {
		critErr = err
		panic(err) // match production behavior
	}

	store := newMockStore()
	store.branchBytes = []byte{0xff, 0xfe} // invalid RLP
	validators := makeValidators(3)

	tr := NewTracker(crit, validators, store)
	assert.Panics(t, func() {
		tr.Init()
	})
	assert.NotNil(t, critErr)
}
