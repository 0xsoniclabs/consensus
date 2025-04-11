package consensus

import "testing"

func TestMetric_StringFormattingConsistent(t *testing.T) {
	metric := Metric{Num: 10, Size: 100}
	if want, got := "{Num=10,Size=100}", metric.String(); want != got {
		t.Fatalf("incorrectly formatted metric string, expected: %s, got: %s", want, got)
	}
}
