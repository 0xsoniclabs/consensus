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
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
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

func setupElection(conn *sql.DB, epoch consensus.Epoch) (*CoreLachesis, *consensustest.TestEventSource, map[consensus.EventHash]*dbEvent, []*dbEvent, error) {
	validators, weights, err := getValidator(conn, epoch)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if len(validators) == 0 {
		return nil, nil, nil, nil, nil
	}

	testLachesis, _, eventStore, _ := NewCoreLachesis(validators, weights)
	testLachesis.store.SwitchGenesis(&consensusstore.Genesis{Epoch: epoch, Validators: testLachesis.store.GetValidators()})

	eventsOrdered, eventMap, err := getEvents(conn, epoch)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return testLachesis, eventStore, eventMap, eventsOrdered, nil
}

func executeElection(testLachesis *CoreLachesis, eventStore *consensustest.TestEventSource, eventsOrdered []*dbEvent) error {
	for _, event := range eventsOrdered {
		if err := ingestEvent(testLachesis, eventStore, event); err != nil {
			return err
		}
	}

	return nil
}

func CheckEpochAgainstDB(conn *sql.DB, epoch consensus.Epoch) error {
	testLachesis, eventStore, eventMap, orderedEvents, err := setupElection(conn, epoch)
	if err != nil {
		return err
	}

	recalculatedAtropoi := make([]consensus.EventHash, 0)
	// Capture the elected atropoi by planting the `applyBlock` callback (nil by default)
	testLachesis.applyBlock = func(block *consensus.Block) *consensus.Validators {
		recalculatedAtropoi = append(recalculatedAtropoi, block.Atropos)
		return nil
	}

	if err := executeElection(testLachesis, eventStore, orderedEvents); err != nil {
		return err
	}

	expectedAtropoi, err := getAtropoi(conn, epoch)
	if err != nil {
		return err
	}
	if want, got := len(expectedAtropoi), len(recalculatedAtropoi); want > got {
		return fmt.Errorf("incorrect number of atropoi recalculated for epoch %d, expected at least: %d, got: %d", epoch, want, got)
	}
	for idx := range expectedAtropoi {
		if want, got := expectedAtropoi[idx], recalculatedAtropoi[idx]; want != got {
			return fmt.Errorf("incorrect atropos for epoch %d on position %d, expected: %s got: %s", epoch, idx, eventMap[want].String(), eventMap[got].String())
		}
	}
	return nil
}

func GetEpochRange(conn *sql.DB) (consensus.Epoch, consensus.Epoch, error) {
	// Query the `Event` table as `Validator` table may include future (empty) epochs
	rows, err := conn.Query(`
		SELECT MIN(e.EpochId), MAX(e.EpochId)
		FROM Event e
	`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

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

func ingestEvent(testLachesis *CoreLachesis, eventStore *consensustest.TestEventSource, event *dbEvent) error {
	testEvent := &consensustest.TestEvent{}
	testEvent.SetSeq(event.seq)
	testEvent.SetCreator(event.validatorId)
	testEvent.SetParents(event.parents)
	testEvent.SetLamport(event.lamportTs)
	testEvent.SetEpoch(testLachesis.store.GetEpoch())
	testEvent.SetID([24]byte(event.hash[8:]))
	eventStore.SetEvent(testEvent)

	return processLocalEvent(testLachesis, testEvent, event.frame)
}

// processLocalEvent simulates a flattened (without redudantant indexing and frame (re)calculations)
// event lifecycle in local computation intensive consensus components - DAG indexing, frame calculation, election
// Conditions and order in which the components are invoked are identical to production Consensus behaviour
func processLocalEvent(testLachesis *CoreLachesis, event *consensustest.TestEvent, targetFrame consensus.Frame) error {
	if err := testLachesis.DagIndexer.Add(event); err != nil {
		return fmt.Errorf("error wihile indexing event: [validator: %d, seq: %d], err: %v", event.Creator(), event.Seq(), err)
	}
	if err := testLachesis.Lachesis.Build(event); err != nil {
		return fmt.Errorf("error wihile building event: [validator: %d, seq: %d], err: %v", event.Creator(), event.Seq(), err)
	}
	if targetFrame != event.Frame() {
		return fmt.Errorf("incorrect frame recalculated for event: [validator: %d, seq: %d], expected: %d, got: %d", event.Creator(), event.Seq(), targetFrame, event.Frame())
	}
	selfParentFrame := testLachesis.getSelfParentFrame(event)
	if selfParentFrame != event.Frame() {
		testLachesis.store.AddRoot(event)
		if err := testLachesis.handleElection(event); err != nil {
			return fmt.Errorf("error wihile processing event: [validator: %d, seq: %d], err: %v", event.Creator(), event.Seq(), err)
		}
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
	defer rows.Close()

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
	defer rows.Close()

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
	defer rows.Close()

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

func getAtropoi(conn *sql.DB, epoch consensus.Epoch) ([]consensus.EventHash, error) {
	rows, err := conn.Query(`
		SELECT e.EventHash
		FROM Atropos a JOIN Event e ON a.AtroposId = e.EventId
		WHERE e.EpochId = ?
		ORDER BY a.AtroposId ASC
	`, epoch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	atropoi := make([]consensus.EventHash, 0)
	for rows.Next() {
		var atroposHashStr string
		err = rows.Scan(&atroposHashStr)
		if err != nil {
			return nil, err
		}

		atroposHash, err := decodeHashStr(atroposHashStr)
		if err != nil {
			return nil, err
		}
		atropoi = append(atropoi, atroposHash)
	}
	return atropoi, nil
}

// hashStr is in hex format, i.e. 0x1a2b3c4d...
func decodeHashStr(hashStr string) (consensus.EventHash, error) {
	hashSlice, err := hex.DecodeString(hashStr[2:])
	if err != nil {
		return consensus.EventHash{}, err
	}
	return consensus.EventHash(hashSlice), nil
}
