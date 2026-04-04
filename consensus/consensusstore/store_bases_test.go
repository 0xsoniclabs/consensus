package consensusstore

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

func TestStoreBases_ConsistentPersistingAndRetrieval_10_10(t *testing.T) {
	testStoreBases_ConsistentPersistingAndRetrieval(t, 10, 10)
}
func TestStoreBases_ConsistentPersistingAndRetrieval_100_20(t *testing.T) {
	testStoreBases_ConsistentPersistingAndRetrieval(t, 100, 20)
}
func TestStoreBases_ConsistentPersistingAndRetrieval_200_30(t *testing.T) {
	testStoreBases_ConsistentPersistingAndRetrieval(t, 200, 30)
}

func TestStoreBases_ConsistentPersistingAndRetrieval_1000_50(t *testing.T) {
	testStoreBases_ConsistentPersistingAndRetrieval(t, 1000, 50)
}

func TestStoreBases_ConsistentPersistingAndRetrieval_1000_100(t *testing.T) {
	testStoreBases_ConsistentPersistingAndRetrieval(t, 1000, 100)
}

func testStoreBases_ConsistentPersistingAndRetrieval(t *testing.T, numFrames int, meanBasesPerFrame int) {
	additionalBasePeriod := consensus.Frame(5)
	store := NewMemStore()
	basesExpected := populateWithBases(t, store, numFrames, meanBasesPerFrame)
	// randomize frame retrieval order
	frameRetrievalOrder := rand.Perm(numFrames)
	for _, f := range frameRetrievalOrder {
		frame := consensus.Frame(f)
		basesRetrieved := simplifyAndSortBases(store.GetFrameBases(frame))
		if !slices.Equal(basesExpected[frame], basesRetrieved) {
			t.Fatalf("unexpected bases retrieved for frame %d, expected: %v, got: %d", frame, basesExpected[frame], basesRetrieved)
		}
		// occasionally persist a base right after retrieving the frame (triggering on-Add cache)
		if frame%additionalBasePeriod == 1 {
			validatorId := consensus.ValidatorID(basesExpected[frame][len(basesExpected[frame])-1]) + 1
			persistBase(store, frame, validatorId)
			basesExpected[frame] = append(basesExpected[frame], validatorId)
			basesRetrieved := simplifyAndSortBases(store.GetFrameBases(frame))
			if !slices.Equal(basesExpected[frame], basesRetrieved) {
				t.Fatalf("unexpected bases retrieved for frame %d, expected: %v, got: %d", frame, basesExpected[frame], basesRetrieved)
			}
		}
	}
}

func populateWithBases(t *testing.T, store *Store, numFrames int, meanBasesPerFrame int) map[consensus.Frame][]consensus.ValidatorID {
	if err := store.OpenEpochDB(1); err != nil {
		t.Fatalf("OpenEpochDB(1) failed")
	}
	basesExpected := make(map[consensus.Frame][]consensus.ValidatorID)
	for i := range meanBasesPerFrame * numFrames {
		// randomize frame insertion order
		frame, validatorID := consensus.Frame(rand.IntN(numFrames)), consensus.ValidatorID(i)
		persistBase(store, frame, validatorID)
		basesExpected[frame] = append(basesExpected[frame], validatorID)
	}
	return basesExpected
}

func simplifyAndSortBases(baseDescriptors []consensus.BaseDescriptor) []consensus.ValidatorID {
	bases := make([]consensus.ValidatorID, 0, len(baseDescriptors))
	for _, descriptor := range baseDescriptors {
		bases = append(bases, descriptor.ValidatorID)
	}
	slices.Sort(bases)
	return bases
}

func persistBase(store *Store, frame consensus.Frame, validatorID consensus.ValidatorID) {
	base := &consensustest.TestEvent{}
	// randomize frame insertion order
	base.SetFrame(frame)
	// identify bases by ValidatorId (convenient as it's part of BaseDescriptor)
	base.SetCreator(validatorID)
	store.AddBase(base)
}
