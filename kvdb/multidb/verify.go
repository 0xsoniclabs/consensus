// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package multidb

import (
	"fmt"
)

func (p *Producer) verifyRecords(oldDBRecords map[DBLocator][]TableRecord) error {
	for oldLoc, records := range oldDBRecords {
		for _, old := range records {
			newRoute := p.RouteOf(old.Req)
			if oldLoc.Type != newRoute.Type {
				return fmt.Errorf("DB type for '%s' doesn't match to previous instance: '%s' != '%s'", old.Req, newRoute.Type, oldLoc.Type)
			}
			if oldLoc.Name != newRoute.Name {
				return fmt.Errorf("DB name for '%s' doesn't match to previous instance: '%s' != '%s'", old.Req, newRoute.Name, oldLoc.Name)
			}
			if old.Table != newRoute.Table {
				return fmt.Errorf("table for '%s' doesn't match to previous instance: '%s' != '%s'", old.Req, newRoute.Table, old.Table)
			}
		}
	}
	return nil
}

func (p *Producer) getRecords() (map[DBLocator][]TableRecord, error) {
	dbRecords := make(map[DBLocator][]TableRecord)
	for typ, producer := range p.allProducers {
		for _, name := range producer.Names() {
			db, err := producer.OpenDB(name)
			if err != nil {
				return nil, fmt.Errorf("failed to open DB %s: %w", name, err)
			}

			var extErr error
			func() {
				defer func() {
					err := db.Close()
					if err != nil {
						extErr = fmt.Errorf("failed to close DB %s/%s: %w", typ, name, err)
					}
				}()

				records, err := ReadTablesList(db, p.tableRecordsKey)
				if err != nil {
					extErr = fmt.Errorf("failed to read tables for %s: %w", name, err)
					return
				}

				locator := DBLocator{
					Type: typ,
					Name: name,
				}
				dbRecords[locator] = records
			}()

			if extErr != nil {
				return nil, extErr
			}
		}
	}
	return dbRecords, nil
}

func (p *Producer) Verify() error {
	dbRecords, err := p.getRecords()
	if err != nil {
		return err
	}
	return p.verifyRecords(dbRecords)
}
