package consensus

import (
	"testing"
)

func TestLog_SetEventName(t *testing.T) {
	eventHash, eventName := EventHash{0}, "event_0"
	SetEventName(eventHash, eventName)
	if got := GetEventName(eventHash); eventName != got {
		t.Fatalf("incorrect event name retrieved, expected: %s, got: %s", eventName, got)
	}
	if want, got := "", GetEventName(EventHash{1}); want != got {
		t.Fatalf("expected empty string but recieved: %s", got)
	}
}

func TestLog_SetNodeName(t *testing.T) {
	nodeID, nodeName := ValidatorID(0), "node_0"
	SetNodeName(nodeID, nodeName)
	if got := GetNodeName(nodeID); nodeName != got {
		t.Fatalf("incorrect event name retrieved, expected: %s, got: %s", nodeName, got)
	}
	if want, got := "", GetNodeName(ValidatorID(1)); want != got {
		t.Fatalf("expected empty string but recieved: %s", got)
	}
}
