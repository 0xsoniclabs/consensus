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

type TypeName string

type Route struct {
	Type   TypeName
	Name   string
	Table  string
	NoDrop bool
}

type scanfRoute struct {
	Name   func(req string) (string, error)
	Type   TypeName
	Table  string
	NoDrop bool
}

type DBLocator struct {
	Type TypeName
	Name string
}

func DBLocatorOf(r Route) DBLocator {
	return DBLocator{
		Type: r.Type,
		Name: r.Name,
	}
}

type TableLocator struct {
	Type  TypeName
	Name  string
	Table string
}

func TableLocatorOf(r Route) TableLocator {
	return TableLocator{
		Type:  r.Type,
		Name:  r.Name,
		Table: r.Table,
	}
}
