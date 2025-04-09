// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

import (
	"strings"
	"testing"
)

func createBaseEvent(epoch Epoch, seq Seq, frame Frame, creator ValidatorID, lamport Lamport, _ int) *BaseEvent {
	mutableEvent := &MutableBaseEvent{}
	mutableEvent.SetEpoch(epoch)
	mutableEvent.SetSeq(seq)
	mutableEvent.SetFrame(frame)
	mutableEvent.SetCreator(creator)
	mutableEvent.SetLamport(lamport)

	var rID [24]byte
	rID[0] = byte(seq)

	return mutableEvent.Build(rID)
}

func TestEventsString(t *testing.T) {
	events := Events{
		createBaseEvent(1, 1, 1, 1, 1, 100),
		createBaseEvent(1, 2, 1, 1, 2, 200),
		createBaseEvent(1, 3, 1, 1, 3, 300),
	}

	result := events.String()

	if result == "" {
		t.Errorf("Events.String() returned empty string")
	}

	for i := range events {
		eventStr := events[i].String()
		if !strings.Contains(result, eventStr) {
			t.Errorf("Events.String() = %s, doesn't contain event %d: %s", result, i, eventStr)
		}
	}

	expectedStrings := make([]string, len(events))
	for i := range events {
		expectedStrings[i] = events[i].String()
	}
	expected := strings.Join(expectedStrings, " ")

	if result != expected {
		t.Errorf("Events.String() = %s, want %s", result, expected)
	}

	emptyEvents := Events{}
	if emptyEvents.String() != "" {
		t.Errorf("Empty Events.String() = %s, want \"\"", emptyEvents.String())
	}
}

func TestEventsMetric(t *testing.T) {
	events := Events{
		createBaseEvent(1, 1, 1, 1, 1, 100),
		createBaseEvent(1, 2, 1, 1, 2, 200),
		createBaseEvent(1, 3, 1, 1, 3, 300),
	}

	metric := events.Metric()
	if metric.Num != 3 {
		t.Errorf("Metric.Num = %d, want 3", metric.Num)
	}

	expectedSize := uint64(0)
	for _, e := range events {
		expectedSize += uint64(e.Size())
	}

	if metric.Size != expectedSize {
		t.Errorf("Metric.Size = %d, want %d", metric.Size, expectedSize)
	}
}

func TestEventsIDs(t *testing.T) {
	events := Events{
		createBaseEvent(1, 1, 1, 1, 1, 100),
		createBaseEvent(1, 2, 1, 1, 2, 200),
	}

	ids := events.IDs()
	if len(ids) != 2 {
		t.Errorf("len(IDs()) = %d, want 2", len(ids))
	}

	for i, event := range events {
		if ids[i] != event.ID() {
			t.Errorf("IDs()[%d] = %v, want %v", i, ids[i], event.ID())
		}
	}
}
