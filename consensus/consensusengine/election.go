// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusengine

import (
	"errors"
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
)

type (
	StronglyReachFn func(a consensus.EventHash, b consensus.EventHash) bool
	GetFrameBasesFn func(f consensus.Frame) []consensusstore.BaseDescriptor
)

// Slot identifies a frame+validator position in the DAG.
type Slot struct {
	Frame     consensus.Frame
	Validator consensus.ValidatorID
}

// BaseAndSlot pairs an event hash with its frame/validator slot.
type BaseAndSlot struct {
	ID   consensus.EventHash
	Slot Slot
}

// ElectionRes is the outcome of a decided election: the certified frame
// and the leader event hash.
type ElectionRes struct {
	Frame  consensus.Frame
	Leader consensus.EventHash
}

type voteID struct {
	fromBase     BaseAndSlot
	forValidator consensus.ValidatorID
}

type voteValue struct {
	decided      bool
	yes          bool
	observedBase consensus.EventHash
}

type election struct {
	frameToCertify consensus.Frame
	validators     *consensus.Validators
	decidedBases   map[consensus.ValidatorID]voteValue
	votes          map[voteID]voteValue
	observe        StronglyReachFn
	getFrameBases  GetFrameBasesFn
}

func NewElection(
	frameToCertify consensus.Frame,
	validators *consensus.Validators,
	stronglyReachFn StronglyReachFn,
	getFrameBases GetFrameBasesFn,
) *election {
	el := &election{
		observe:       stronglyReachFn,
		getFrameBases: getFrameBases,
	}
	el.Reset(validators, frameToCertify)
	return el
}

func (el *election) Reset(validators *consensus.Validators, frameToCertify consensus.Frame) {
	el.validators = validators
	el.frameToCertify = frameToCertify
	el.votes = make(map[voteID]voteValue)
	el.decidedBases = make(map[consensus.ValidatorID]voteValue)
}

func (el *election) ResetEpoch(frameToCertify consensus.Frame, validators *consensus.Validators) {
	el.Reset(validators, frameToCertify)
}

func (el *election) ProcessBase(newBase BaseAndSlot) (*ElectionRes, error) {
	res, err := el.chooseLeader()
	if err != nil || res != nil {
		return res, err
	}

	if newBase.Slot.Frame <= el.frameToCertify {
		return nil, nil
	}
	round := newBase.Slot.Frame - el.frameToCertify
	if round == 0 {
		return nil, nil
	}

	notDecided := el.notDecidedBases()

	var observedBases []BaseAndSlot
	var observedBasesMap map[consensus.ValidatorID]BaseAndSlot
	if round == 1 {
		observedBasesMap = el.observedBasesMap(newBase.ID, newBase.Slot.Frame-1)
	} else {
		observedBases = el.observedBasesList(newBase.ID, newBase.Slot.Frame-1)
	}

	for _, validatorSubject := range notDecided {
		vote := voteValue{}

		if round == 1 {
			observed, ok := observedBasesMap[validatorSubject]
			vote.yes = ok
			vote.decided = false
			if ok {
				vote.observedBase = observed.ID
			}
		} else {
			yesVotes := el.validators.NewCounter()
			noVotes := el.validators.NewCounter()
			allVotes := el.validators.NewCounter()

			var subjectHash *consensus.EventHash
			for _, observed := range observedBases {
				vid := voteID{
					fromBase:     observed,
					forValidator: validatorSubject,
				}

				if v, ok := el.votes[vid]; ok {
					if v.yes && subjectHash != nil && *subjectHash != v.observedBase {
						return nil, fmt.Errorf(
							"strongly-reached by 2 fork bases => more than 1/3W are Byzantine (%s != %s, election frame=%d, validator=%d)",
							subjectHash.String(), v.observedBase.String(), el.frameToCertify, validatorSubject)
					}

					if v.yes {
						subjectHash = &v.observedBase
						yesVotes.CountVoteByID(observed.Slot.Validator)
					} else {
						noVotes.CountVoteByID(observed.Slot.Validator)
					}
					if !allVotes.CountVoteByID(observed.Slot.Validator) {
						return nil, fmt.Errorf(
							"strongly-reached by 2 fork bases => more than 1/3W are Byzantine (election frame=%d, validator=%d)",
							el.frameToCertify, validatorSubject)
					}
				} else {
					return nil, errors.New("every base must vote for every not decided subject. possibly bases are processed out of order")
				}
			}
			if !allVotes.QuorumReached() {
				return nil, errors.New("base must be strongly-reached by at least 2/3W of prev bases. possibly bases are processed out of order")
			}

			vote.yes = yesVotes.Sum() >= noVotes.Sum()
			if vote.yes && subjectHash != nil {
				vote.observedBase = *subjectHash
			}

			vote.decided = yesVotes.QuorumReached() || noVotes.QuorumReached()
			if vote.decided {
				el.decidedBases[validatorSubject] = vote
			}
		}
		vid := voteID{
			fromBase:     newBase,
			forValidator: validatorSubject,
		}
		el.votes[vid] = vote
	}

	return el.chooseLeader()
}

func (el *election) chooseLeader() (*ElectionRes, error) {
	for _, validator := range el.validators.SortedIDs() {
		vote, ok := el.decidedBases[validator]
		if !ok {
			return nil, nil
		}
		if vote.yes {
			return &ElectionRes{
				Frame:  el.frameToCertify,
				Leader: vote.observedBase,
			}, nil
		}
	}
	return nil, errors.New("all bases decided as 'no', which is possible only if more than 1/3W are Byzantine")
}

func (el *election) notDecidedBases() []consensus.ValidatorID {
	result := make([]consensus.ValidatorID, 0, el.validators.Len())
	for _, validator := range el.validators.IDs() {
		if _, ok := el.decidedBases[validator]; !ok {
			result = append(result, validator)
		}
	}
	return result
}

func (el *election) observedBasesMap(base consensus.EventHash, frame consensus.Frame) map[consensus.ValidatorID]BaseAndSlot {
	result := make(map[consensus.ValidatorID]BaseAndSlot, el.validators.Len())
	frameBases := el.getFrameBases(frame)
	for _, fb := range frameBases {
		if el.observe(base, fb.BaseHash) {
			if _, exists := result[fb.ValidatorID]; exists {
				panic(fmt.Sprintf("equivocation detected: multiple observed bases for validator %v in frame %v", fb.ValidatorID, frame))
			}
			result[fb.ValidatorID] = BaseAndSlot{
				ID: fb.BaseHash,
				Slot: Slot{
					Frame:     frame,
					Validator: fb.ValidatorID,
				},
			}
		}
	}
	return result
}

func (el *election) observedBasesList(base consensus.EventHash, frame consensus.Frame) []BaseAndSlot {
	result := make([]BaseAndSlot, 0, el.validators.Len())
	frameBases := el.getFrameBases(frame)
	for _, fb := range frameBases {
		if el.observe(base, fb.BaseHash) {
			result = append(result, BaseAndSlot{
				ID: fb.BaseHash,
				Slot: Slot{
					Frame:     frame,
					Validator: fb.ValidatorID,
				},
			})
		}
	}
	return result
}
