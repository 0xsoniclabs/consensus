// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package flushable

type closeDropWrapped struct {
	*LazyFlushable
	close func() error
	drop  func()
}

func (w *closeDropWrapped) Close() error {
	return w.close()
}

func (w *closeDropWrapped) RealClose() error {
	return w.LazyFlushable.Close()
}

func (w *closeDropWrapped) Drop() {
	w.drop()
}

func (w *closeDropWrapped) RealDrop() {
	w.LazyFlushable.Drop()
}

func (w *closeDropWrapped) AncientDatadir() (string, error) {
	return "", nil
}
