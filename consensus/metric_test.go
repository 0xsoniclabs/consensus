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

import "testing"

func TestMetric_StringFormattingConsistent(t *testing.T) {
	metric := Metric{Num: 10, Size: 100}
	if want, got := "{Num=10,Size=100}", metric.String(); want != got {
		t.Fatalf("incorrectly formatted metric string, expected: %s, got: %s", want, got)
	}
}
