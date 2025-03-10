// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package pebble

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/0xsoniclabs/kvdb"
)

type Producer struct {
	datadir         string
	getCacheFdLimit func(string) (int, int)
}

// NewProducer of Pebble db.
func NewProducer(datadir string, getCacheFdLimit func(string) (int, int)) kvdb.IterableDBProducer {
	return &Producer{
		datadir:         datadir,
		getCacheFdLimit: getCacheFdLimit,
	}
}

// Names of existing databases.
func (p *Producer) Names() []string {
	files, err := ioutil.ReadDir(p.datadir)
	if err != nil {
		panic(err)
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		names = append(names, f.Name())
	}
	return names
}

// OpenDB or create db with name.
func (p *Producer) OpenDB(name string) (kvdb.Store, error) {
	path := p.resolvePath(name)

	err := os.MkdirAll(path, 0700)
	if err != nil {
		return nil, err
	}

	onDrop := func() {
		_ = os.RemoveAll(path)
	}

	cache, fdlimit := p.getCacheFdLimit(name)
	db, err := New(path, cache, fdlimit, nil, onDrop)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (p *Producer) resolvePath(name string) string {
	return filepath.Join(p.datadir, name)
}
