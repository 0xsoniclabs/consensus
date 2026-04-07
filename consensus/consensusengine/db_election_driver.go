// Copyright (c) 2026 Sonic Operations Ltd

package consensusengine

import (
	"database/sql"
	"encoding/hex"
	"errors"
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

	testLachesis, _, eventStore, _ := NewBootstrappedCoreConsensus(validators, weights)
	if err := testLachesis.store.SwitchGenesis(&consensusstore.Genesis{Epoch: epoch, Validators: testLachesis.store.GetValidators()}); err != nil {
		return nil, nil, nil, nil, err
	}

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

	recalculatedLeaders := make([]consensus.EventHash, 0)
	// Capture the elected leaders by planting the `applyBlock` callback (nil by default)
	testLachesis.applyBlock = func(block *consensus.Block) *consensus.Validators {
		recalculatedLeaders = append(recalculatedLeaders, block.Leader)
		return nil
	}

	if err := executeElection(testLachesis, eventStore, orderedEvents); err != nil {
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

func GetEpochRange(conn *sql.DB) (consensus.Epoch, consensus.Epoch, error) {
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

func ingestEvent(testLachesis *CoreLachesis, eventStore *consensustest.TestEventSource, event *dbEvent) error {
	testEvent := &consensustest.TestEvent{}
	testEvent.SetFrame(event.frame)
	testEvent.SetSeq(event.seq)
	testEvent.SetCreator(event.validatorId)
	testEvent.SetParents(event.parents)
	testEvent.SetLamport(event.lamportTs)
	testEvent.SetEpoch(testLachesis.store.GetEpoch())
	testEvent.SetID([24]byte(event.hash[8:]))
	eventStore.SetEvent(testEvent)

	return processLocalEvent(testLachesis, testEvent)
}

// processLocalEvent simulates a flattened (without redudantant indexing and frame (re)calculations)
// event lifecycle in local computation intensive consensus components - DAG indexing, frame calculation, election
// Conditions and order in which the components are invoked are identical to production Consensus behaviour
func processLocalEvent(testLachesis *CoreLachesis, event *consensustest.TestEvent) error {
	if err := testLachesis.DagIndexer.Add(event); err != nil {
		return fmt.Errorf("error wihile indexing event: [validator: %d, seq: %d], err: %v", event.Creator(), event.Seq(), err)
	}
	if err := testLachesis.Lachesis.Process(event); err != nil {
		return fmt.Errorf("error while processing event: [validator: %d, seq: %d], err: %v", event.Creator(), event.Seq(), err)
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
