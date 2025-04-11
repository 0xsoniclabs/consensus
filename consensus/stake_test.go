package consensus

import (
	"math/rand/v2"
	"testing"
)

func TestStakeCounting_ConsistentSumAndQuorumReached(t *testing.T) {
	validators, weightCounter := validatorsAndCounterSample(100)
	sumWeight := Weight(0)
	quorumWeight := validators.Quorum()
	for _, validatorID := range validators.IDs() {
		// Alternate for variety
		if validatorID%2 == 0 {
			weightCounter.CountVoteByIndex(validators.GetIdx(validatorID))
		} else {
			weightCounter.CountVoteByID(validatorID)
		}
		if validatorID%5 == 0 {
			// duplicate vote for every 5th - should not affect the voting process
			weightCounter.CountVoteByID(validatorID)
		}
		sumWeight += validators.GetWeightByIdx(validators.GetIdx(validatorID))
		if got := weightCounter.Sum(); sumWeight != got {
			t.Fatalf("incosistent weight sum while voting, expected: %d, got: %d", sumWeight, got)
		}
		if want, got := sumWeight >= quorumWeight, weightCounter.HasQuorum(); want != got {
			t.Fatalf("unexpected quorum achieved status, quorumWeight := %d, collected weight: %d, expected: %t, got: %t", quorumWeight, sumWeight, want, got)
		}
	}
}

func TestStakeCounting_VotesAlreadyCounted(t *testing.T) {
	validators, weightCounter := validatorsAndCounterSample(100)

	for _, validatorID := range validators.IDs() {
		// Count even IDs through ID and odd ones through Index
		var notCountedBefore bool
		if validatorID%2 == 0 {
			notCountedBefore = weightCounter.CountVoteByID(validatorID)
		} else {
			notCountedBefore = weightCounter.CountVoteByIndex(validators.GetIdx(validatorID))
		}
		if !notCountedBefore {
			t.Errorf("notCountedBefore expected to be false for ValidatorID: %d", validatorID)
		}
	}

	for _, validatorID := range validators.IDs() {
		// Invert for verification
		var notCountedBefore bool
		if validatorID%2 == 0 {
			notCountedBefore = weightCounter.CountVoteByIndex(validators.GetIdx(validatorID))
		} else {
			notCountedBefore = weightCounter.CountVoteByID(validatorID)
		}
		if notCountedBefore {
			t.Errorf("notCountedBefore expected to be true for ValidatorID: %d", validatorID)
		}
	}
}

func TestStake_NumCountedConsistent(t *testing.T) {
	validators, weightCounter := validatorsAndCounterSample(100)
	pickedValidators := validators.IDs()
	rand.Shuffle(len(pickedValidators), func(i, j int) { pickedValidators[i], pickedValidators[j] = pickedValidators[j], pickedValidators[i] })
	pickedValidators = pickedValidators[0 : len(pickedValidators)/3]

	for _, validatorID := range pickedValidators {
		// Alternate for variety
		if validatorID%2 == 0 {
			weightCounter.CountVoteByIndex(validators.GetIdx(validatorID))
		} else {
			weightCounter.CountVoteByID(validatorID)
		}
	}
	if want, got := len(pickedValidators), weightCounter.NumCounted(); want != got {
		t.Fatalf("unexpected total number of votes, expected: %d, got: %d", want, got)
	}
}

func validatorsAndCounterSample(sampleSize int) (*Validators, *WeightCounter) {
	builder := NewValidatorsBuilder()
	for i := range sampleSize {
		builder.Set(ValidatorID(i), Weight(i*5))
	}
	validators := builder.Build()
	weightCounter := validators.NewCounter()
	return validators, weightCounter
}
