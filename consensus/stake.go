// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

type (
	// Weight amount.
	Weight uint32
)

type (
	// WeightCounterProvider providers weight counter.
	WeightCounterProvider func() *WeightCounter

	// WeightCounter counts weights.
	WeightCounter struct {
		validators   Validators
		alreadyVoted []bool // ValidatorIdx -> bool

		quorum     Weight
		antiQuorum Weight
		sum        Weight
		antiSum    Weight
	}
)

func (vv Validators) NewCounter() *WeightCounter {
	return newWeightCounter(vv)
}

func newWeightCounter(vv Validators) *WeightCounter {
	return &WeightCounter{
		validators:   vv,
		quorum:       vv.Quorum(),
		antiQuorum:   vv.TotalWeight() - vv.Quorum(),
		alreadyVoted: make([]bool, vv.Len()),
		sum:          0,
		antiSum:      0,
	}
}

func (s *WeightCounter) CountVoteByID(v ValidatorID) bool {
	validatorIdx := s.validators.GetIdx(v)
	return s.CountVoteByIndex(validatorIdx)
}

// CountVoteByIndex increases the vote sum by validator's weight if the validator has not voted before, returning the status of the vote
func (s *WeightCounter) CountVoteByIndex(validatorIdx ValidatorIndex) bool {
	if s.alreadyVoted[validatorIdx] {
		return false
	}
	s.alreadyVoted[validatorIdx] = true

	s.sum += s.validators.GetWeightByIdx(validatorIdx)
	return true
}

func (s *WeightCounter) CountAntiVoteByID(v ValidatorID) bool {
	validatorIdx := s.validators.GetIdx(v)
	return s.CountAntiVoteByIndex(validatorIdx)
}

// CountVoteByIndex increases the anti vote sum by validator's weight if the validator has not voted before, returning the status of the vote
func (s *WeightCounter) CountAntiVoteByIndex(validatorIdx ValidatorIndex) bool {
	if s.alreadyVoted[validatorIdx] {
		return false
	}
	s.alreadyVoted[validatorIdx] = true

	s.antiSum += s.validators.GetWeightByIdx(validatorIdx)
	return true
}

func (s *WeightCounter) QuorumReached() bool {
	return s.sum >= s.quorum
}

// Anti Quorum denotes the weighted sum of validators that didn't vote towards the quorum.
// It enables reaching early quorum decisions when the threshold (TotalValidatorWeight - Quorum) has been surpassed,
// leveraging the fact that the quorum can't be reached no matter how validators vote in future.
func (s *WeightCounter) AntiQuorumReached() bool {
	return s.antiSum > s.antiQuorum
}

func (s *WeightCounter) Sum() Weight {
	return s.sum
}

func (s *WeightCounter) AntiSum() Weight {
	return s.antiSum
}

func (s *WeightCounter) NumCounted() int {
	num := 0
	for _, counted := range s.alreadyVoted {
		if counted {
			num++
		}
	}
	return num
}
