// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package table

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/consensus/kvdb"
)

type testTables struct {
	NoTable interface{}
	Manual  kvdb.Store `table:"-"`
	Nil     kvdb.Store `table:"-"`
	Auto1   kvdb.Store `table:"A"`
	Auto2   kvdb.Store `table:"B"`
	Auto3   kvdb.Store `table:"C"`
}

func TestOpenTables(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	prefix := "prefix"

	mockStore := func() kvdb.Store {
		store := NewMockStore(ctrl)
		store.EXPECT().Close().
			Times(1).
			Return(nil)
		return store
	}

	dbs := NewMockDBProducer(ctrl)
	dbs.EXPECT().OpenDB(gomock.Any()).
		Times(3).
		DoAndReturn(func(name string) (kvdb.Store, error) {
			require.Contains(name, prefix)
			return mockStore(), nil
		})

	tt := &testTables{}

	// open auto
	err := OpenTables(tt, dbs, prefix)
	require.NoError(err)
	require.NotNil(tt.Auto1)
	require.NotNil(tt.Auto2)
	require.NotNil(tt.Auto3)
	require.Nil(tt.NoTable)
	require.Nil(tt.Nil)

	// open manual
	require.Nil(tt.Manual)
	tt.Manual = mockStore()
	require.NotNil(tt.Manual)

	// close all
	err = CloseTables(tt)
	require.NoError(err)
	require.NotNil(tt.Auto1)
	require.NotNil(tt.Auto2)
	require.NotNil(tt.Auto3)
	require.Nil(tt.NoTable)
	require.Nil(tt.Nil)
	require.NotNil(tt.Manual)
}
