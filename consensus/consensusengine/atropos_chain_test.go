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
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestRegressionData_FantomNetwork(t *testing.T) {
	testRegressionData(t, "testdata/events-5577.db")
}

func TestRegressionData_SonicNetwork(t *testing.T) {
	testRegressionData(t, "testdata/events-8000-partial.db")
}

func BenchmarkElectionFantomNetwork(b *testing.B) {
	benchmarkElection(b, "testdata/events-5577.db")
}

func BenchmarkElectionSonicNetwork(b *testing.B) {
	benchmarkElection(b, "testdata/events-8000-partial.db")
}

func testRegressionData(t *testing.T, dbPath string) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	epochMin, epochMax, err := GetEpochRange(conn)
	if err != nil {
		t.Fatal(err)
	}
	for epoch := epochMin; epoch <= epochMax; epoch++ {
		if err := CheckEpochAgainstDB(conn, epoch); err != nil {
			t.Fatal(err)
		}
	}
}

func benchmarkElection(b *testing.B, dbPath string) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	epochMin, epochMax, err := GetEpochRange(conn)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for range b.N {
		for epoch := epochMin; epoch <= epochMax; epoch++ {
			b.StopTimer()
			testLachesis, eventStore, _, orderedEvents, err := setupElection(conn, epoch)
			if err != nil {
				b.Fatal(err)
			}

			b.StartTimer()
			if err := executeElection(testLachesis, eventStore, orderedEvents); err != nil {
				b.Fatal(err)
			}
		}
	}
}
