// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package main

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusengine"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

type dbEvent struct {
	hash        consensus.EventHash
	validatorId consensus.ValidatorID
	seq         consensus.Seq
	frame       consensus.Frame
	lamportTs   consensus.Lamport
	parents     []consensus.EventHash
}

func (e *dbEvent) String() string {
	return fmt.Sprintf("{Epoch:%d Validator:%d Frame:%d Seq:%d Lamport:%d}", e.hash.Epoch(), e.validatorId, e.frame, e.seq, e.lamportTs)
}

func newConsensusEngine(validators []consensus.ValidatorID, weights []consensus.Weight) (*consensusengine.IndexedLachesis, *consensusstore.Store, *consensustest.TestEventSource) {
	validatorsMap := make(consensus.ValidatorsBuilder, len(validators))
	for i, v := range validators {
		if weights == nil {
			validatorsMap[v] = 1
		} else {
			validatorsMap[v] = weights[i]
		}
	}
	store := consensusstore.NewMemStore()
	if err := store.ApplyGenesis(&consensusstore.Genesis{
		Validators: validatorsMap.Build(),
		Epoch:      consensus.FirstEpoch,
	}); err != nil {
		panic(err)
	}
	input := consensustest.NewTestEventSource()
	crit := func(err error) { panic(err) }
	dagIdx := dagindexer.NewIndex(crit, dagindexer.LiteConfig())
	engine := consensusengine.NewIndexedLachesis(store, input, dagIdx, crit, consensusengine.DefaultConfig())
	return engine, store, input
}



func executeElection(engine *consensusengine.IndexedLachesis, store *consensusstore.Store, eventStore *consensustest.TestEventSource, eventsOrdered []*dbEvent) error {
	for _, event := range eventsOrdered {
		if err := ingestEvent(engine, store, eventStore, event); err != nil {
			return err
		}
	}
	return nil
}

func checkEpochAgainstDB(conn *sql.DB, epoch consensus.Epoch) error {
	validators, weights, err := getValidator(conn, epoch)
	if err != nil {
		return err
	}
	if len(validators) == 0 {
		return nil
	}

	engine, store, eventStore := newConsensusEngine(validators, weights)

	recalculatedLeaders := make([]consensus.EventHash, 0)

	if err := engine.Bootstrap(consensus.ConsensusCallbacks{
		BeginBlock: func(block *consensus.Block) consensus.BlockCallbacks {
			return consensus.BlockCallbacks{
				EndBlock: func() (sealEpoch *consensus.Validators) {
					recalculatedLeaders = append(recalculatedLeaders, block.Leader)
					return nil
				},
			}
		},
	}); err != nil {
		return err
	}

	if err := engine.Reset(epoch, store.GetValidators()); err != nil {
		return err
	}

	eventsOrdered, eventMap, err := getEvents(conn, epoch)
	if err != nil {
		return err
	}

	if err := executeElection(engine, store, eventStore, eventsOrdered); err != nil {
		return err
	}

	expectedLeaders, err := getLeaders(conn, epoch)
	if err != nil {
		return err
	}
	if want, got := len(expectedLeaders), len(recalculatedLeaders); want > got {
		return fmt.Errorf("incorrect number of leaders recalculated for epoch %d, expected at least: %d, got: %d", epoch, want, got)
	}
	for idx := range expectedLeaders {
		if want, got := expectedLeaders[idx], recalculatedLeaders[idx]; want != got {
			return fmt.Errorf("incorrect leader for epoch %d on position %d, expected: %s got: %s", epoch, idx, eventMap[want].String(), eventMap[got].String())
		}
	}
	return nil
}

func getEpochRange(conn *sql.DB) (consensus.Epoch, consensus.Epoch, error) {
	// Query the `Event` table as `Validator` table may include future (empty) epochs
	rows, err := conn.Query(`
		SELECT MIN(e.EpochId), MAX(e.EpochId)
		FROM Event e
	`)
	if err != nil {
		return 0, 0, err
	}
	defer closeRowsAndCombineErrors(&err, rows)

	var epochMin, epochMax consensus.Epoch
	if !rows.Next() {
		return 0, 0, fmt.Errorf("no non-empty epochs in database")
	}
	err = rows.Scan(&epochMin, &epochMax)
	if err != nil {
		return 0, 0, err
	}
	return epochMin, epochMax, nil
}

func ingestEvent(engine *consensusengine.IndexedLachesis, store *consensusstore.Store, eventStore *consensustest.TestEventSource, event *dbEvent) error {
	testEvent := &consensustest.TestEvent{}
	testEvent.SetFrame(event.frame)
	testEvent.SetSeq(event.seq)
	testEvent.SetCreator(event.validatorId)
	testEvent.SetParents(event.parents)
	testEvent.SetLamport(event.lamportTs)
	testEvent.SetEpoch(store.GetEpoch())
	testEvent.SetID([24]byte(event.hash[8:]))
	eventStore.SetEvent(testEvent)

	// Simulates a flattened (without redundant indexing and frame recalculations)
	// event lifecycle. DagIndexer.Add + Lachesis.Process are invoked separately
	// to skip the frame recalculation that IndexedLachesis.Process would do.
	if err := engine.DagIndexer.Add(testEvent); err != nil {
		return fmt.Errorf("error while indexing event: [validator: %d, seq: %d], err: %v", event.validatorId, event.seq, err)
	}
	if err := engine.Lachesis.Process(testEvent); err != nil {
		return fmt.Errorf("error while processing event: [validator: %d, seq: %d], err: %v", event.validatorId, event.seq, err)
	}
	return nil
}

func getValidator(conn *sql.DB, epoch consensus.Epoch) ([]consensus.ValidatorID, []consensus.Weight, error) {
	rows, err := conn.Query(`
		SELECT ValidatorId, Weight
		FROM Validator
		WHERE EpochId = ?
	`, epoch)
	if err != nil {
		return nil, nil, err
	}
	defer closeRowsAndCombineErrors(&err, rows)

	validators := make([]consensus.ValidatorID, 0)
	weights := make([]consensus.Weight, 0)
	for rows.Next() {
		var validatorId consensus.ValidatorID
		var weight consensus.Weight

		err = rows.Scan(&validatorId, &weight)
		if err != nil {
			return nil, nil, err
		}

		validators = append(validators, validatorId)
		weights = append(weights, weight)
	}
	return validators, weights, nil
}

func getEvents(conn *sql.DB, epoch consensus.Epoch) ([]*dbEvent, map[consensus.EventHash]*dbEvent, error) {
	rows, err := conn.Query(`
		SELECT e.EventHash, e.ValidatorId, e.SequenceNumber, e.FrameId, e.LamportNumber
		FROM Event e
		WHERE e.EpochId = ?
		ORDER BY e.LamportNumber ASC
	`, epoch)
	if err != nil {
		return nil, nil, err
	}
	defer closeRowsAndCombineErrors(&err, rows)

	eventMap := make(map[consensus.EventHash]*dbEvent)
	eventsOrdered := make([]*dbEvent, 0)
	for rows.Next() {
		var hashStr string
		var validatorId consensus.ValidatorID
		var seq consensus.Seq
		var frame consensus.Frame
		var lamportTs consensus.Lamport
		err = rows.Scan(&hashStr, &validatorId, &seq, &frame, &lamportTs)
		if err != nil {
			return nil, nil, err
		}

		eventHash, err := decodeHashStr(hashStr)
		if err != nil {
			return nil, nil, err
		}
		event := &dbEvent{
			hash:        eventHash,
			validatorId: validatorId,
			seq:         seq,
			frame:       frame,
			lamportTs:   lamportTs,
			parents:     make([]consensus.EventHash, 0),
		}
		eventsOrdered = append(eventsOrdered, event)
		eventMap[eventHash] = event
	}
	return eventsOrdered, eventMap, appointParents(conn, eventMap, epoch)
}

func appointParents(conn *sql.DB, eventMap map[consensus.EventHash]*dbEvent, epoch consensus.Epoch) error {
	rows, err := conn.Query(`
		SELECT e.EventHash, eParent.EventHash
		FROM Event e JOIN Parent p ON e.EventId = p.EventId JOIN Event eParent ON eParent.EventId = p.ParentId
		WHERE e.EpochId = ?
	`, epoch)
	if err != nil {
		return err
	}
	defer closeRowsAndCombineErrors(&err, rows)

	for rows.Next() {
		var eventHashStr string
		var parentHashStr string
		err = rows.Scan(&eventHashStr, &parentHashStr)
		if err != nil {
			return err
		}

		eventHash, err := decodeHashStr(eventHashStr)
		if err != nil {
			return err
		}
		parentHash, err := decodeHashStr(parentHashStr)
		if err != nil {
			return err
		}
		event, ok := eventMap[eventHash]
		if !ok {
			return fmt.Errorf(
				"incomplete events.db - child event not found. epoch: %d, child event: %s, parent event: %s",
				epoch,
				eventHash,
				parentHash,
			)
		}
		if _, ok := eventMap[parentHash]; !ok {
			return fmt.Errorf(
				"incomplete events.db - parent event not found. epoch: %d, child event: %s, parent event: %s",
				epoch,
				eventHash,
				parentHash,
			)
		}
		event.parents = append(event.parents, parentHash)
		// ensure the self parent is first in the slice
		if eventMap[parentHash].validatorId == event.validatorId {
			event.parents[0], event.parents[len(event.parents)-1] = event.parents[len(event.parents)-1], event.parents[0]
		}
	}
	return nil
}

func getLeaders(conn *sql.DB, epoch consensus.Epoch) ([]consensus.EventHash, error) {
	rows, err := conn.Query(`
		SELECT e.EventHash
		FROM Atropos a JOIN Event e ON a.AtroposId = e.EventId
		WHERE e.EpochId = ?
		ORDER BY a.AtroposId ASC
	`, epoch)
	if err != nil {
		return nil, err
	}
	defer closeRowsAndCombineErrors(&err, rows)

	leaders := make([]consensus.EventHash, 0)
	for rows.Next() {
		var leaderHashStr string
		err = rows.Scan(&leaderHashStr)
		if err != nil {
			return nil, err
		}

		leaderHash, err := decodeHashStr(leaderHashStr)
		if err != nil {
			return nil, err
		}
		leaders = append(leaders, leaderHash)
	}
	return leaders, nil
}

// hashStr is in hex format, i.e. 0x1a2b3c4d...
func decodeHashStr(hashStr string) (consensus.EventHash, error) {
	hashSlice, err := hex.DecodeString(hashStr[2:])
	if err != nil {
		return consensus.EventHash{}, err
	}
	return consensus.EventHash(hashSlice), nil
}

func closeRowsAndCombineErrors(errPtr *error, rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		*errPtr = errors.Join(*errPtr, err)
	}
}
