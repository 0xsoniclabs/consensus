package consensusstore

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

func TestStoreRoots_ConsistentPersistingAndRetrieval_10_10(t *testing.T) {
	testStoreRoots_ConsistentPersistingAndRetrieval(t, 10, 10)
}
func TestStoreRoots_ConsistentPersistingAndRetrieval_100_20(t *testing.T) {
	testStoreRoots_ConsistentPersistingAndRetrieval(t, 100, 20)
}
func TestStoreRoots_ConsistentPersistingAndRetrieval_200_30(t *testing.T) {
	testStoreRoots_ConsistentPersistingAndRetrieval(t, 200, 30)
}

func TestStoreRoots_ConsistentPersistingAndRetrieval_1000_50(t *testing.T) {
	testStoreRoots_ConsistentPersistingAndRetrieval(t, 1000, 50)
}

func TestStoreRoots_ConsistentPersistingAndRetrieval_1000_100(t *testing.T) {
	testStoreRoots_ConsistentPersistingAndRetrieval(t, 1000, 100)
}

func testStoreRoots_ConsistentPersistingAndRetrieval(t *testing.T, numFrames int, meanRootsPerFrame int) {
	additionalRootPeriod := consensus.Frame(5)
	store := NewMemStore()
	rootsExpected := populateWithRoots(t, store, numFrames, meanRootsPerFrame)
	// randomize frame retrieval order
	frameRetrievalOrder := rand.Perm(numFrames)
	for _, f := range frameRetrievalOrder {
		frame := consensus.Frame(f)
		rootsRetrieved := simplifyAndSortRoots(store.GetFrameRoots(frame))
		if !slices.Equal(rootsExpected[frame], rootsRetrieved) {
			t.Fatalf("unexpected roots retrieved for frame %d, expected: %v, got: %d", frame, rootsExpected[frame], rootsRetrieved)
		}
		// occasionally persist a root right after retrieving the frame (triggering on-Add cache)
		if frame%additionalRootPeriod == 1 {
			validatorId := consensus.ValidatorID(rootsExpected[frame][len(rootsExpected[frame])-1]) + 1
			persistRoot(store, frame, validatorId)
			rootsExpected[frame] = append(rootsExpected[frame], validatorId)
			rootsRetrieved := simplifyAndSortRoots(store.GetFrameRoots(frame))
			if !slices.Equal(rootsExpected[frame], rootsRetrieved) {
				t.Fatalf("unexpected roots retrieved for frame %d, expected: %v, got: %d", frame, rootsExpected[frame], rootsRetrieved)
			}
		}
	}
}

func populateWithRoots(t *testing.T, store *Store, numFrames int, meanRootsPerFrame int) map[consensus.Frame][]consensus.ValidatorID {
	if err := store.OpenEpochDB(1); err != nil {
		t.Fatalf("OpenEpochDB(1) failed")
	}
	rootsExpected := make(map[consensus.Frame][]consensus.ValidatorID)
	for i := range meanRootsPerFrame * numFrames {
		// randomize frame insertion order
		frame, validatorID := consensus.Frame(rand.IntN(numFrames)), consensus.ValidatorID(i)
		persistRoot(store, frame, validatorID)
		rootsExpected[frame] = append(rootsExpected[frame], validatorID)
	}
	return rootsExpected
}

func simplifyAndSortRoots(rootDescriptors []RootDescriptor) []consensus.ValidatorID {
	roots := make([]consensus.ValidatorID, 0, len(rootDescriptors))
	for _, descriptor := range rootDescriptors {
		roots = append(roots, descriptor.ValidatorID)
	}
	slices.Sort(roots)
	return roots
}

func persistRoot(store *Store, frame consensus.Frame, validatorID consensus.ValidatorID) {
	root := &consensustest.TestEvent{}
	// randomize frame insertion order
	root.SetFrame(frame)
	// identify roots by ValidatorId (convenient as it's part of RootDescriptor)
	root.SetCreator(validatorID)
	store.AddRoot(root)
}
