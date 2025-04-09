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
	"fmt"
	"os"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusengine"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
)

var (
	DbPathFlag = cli.StringFlag{
		Name:     "db",
		Usage:    "sqlite3 event db path",
		Required: true,
	}
	EpochMinFlag = cli.UintFlag{
		Name:  "epoch.min",
		Usage: "Lower bound (inclusive) for epochs to be checked",
	}
	EpochMaxFlag = cli.UintFlag{
		Name:  "epoch.max",
		Usage: "Upper bound (inclusive) for epochs to be checked",
	}
)

func main() {
	app := &cli.App{
		Name:        "Event DB Checker",
		Description: "Consensus regression testing tool",
		Copyright:   "(c) 2025 Sonic Labs",
		Flags:       []cli.Flag{&DbPathFlag, &EpochMinFlag, &EpochMaxFlag},
		Action:      run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", ctx.String(DbPathFlag.Name)))
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "error closing database connection: %v\n", closeErr)
		}
	}()
	if err := conn.Ping(); err != nil {
		return err
	}

	epochMin, epochMax, err := consensusengine.GetEpochRange(conn)
	if err != nil {
		return err
	}
	if ctx.IsSet(EpochMinFlag.Name) {
		epochMin = max(epochMin, consensus.Epoch(ctx.Uint(EpochMinFlag.Name)))
	}
	if ctx.IsSet(EpochMaxFlag.Name) {
		epochMax = min(epochMax, consensus.Epoch(ctx.Uint(EpochMaxFlag.Name)))
	}
	if epochMin > epochMax {
		return fmt.Errorf("invalid range of epochs requested: [%d, %d]", epochMin, epochMax)
	}

	for epoch := epochMin; epoch <= epochMax; epoch++ {
		if err := consensusengine.CheckEpochAgainstDB(conn, epoch); err != nil {
			return err
		}
	}
	return nil
}
