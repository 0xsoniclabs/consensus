// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: <TBD>
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package workers

import (
	"errors"
	"sync"
)

var (
	errTerminated = errors.New("terminated")
)

type Workers struct {
	quit  chan struct{}
	wg    *sync.WaitGroup
	tasks chan func()
}

func New(wg *sync.WaitGroup, quit chan struct{}, maxTasks int) *Workers {
	return &Workers{
		tasks: make(chan func(), maxTasks),
		quit:  quit,
		wg:    wg,
	}
}

func (w *Workers) Start(workersN int) {
	for i := 0; i < workersN; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			worker(w.tasks, w.quit)
		}()
	}
}

func (w *Workers) Enqueue(fn func()) error {
	select {
	case w.tasks <- fn:
		return nil
	case <-w.quit:
		return errTerminated
	}
}

func (w *Workers) Drain() {
	for {
		select {
		case <-w.tasks:
			continue
		default:
			return
		}
	}
}

func (w *Workers) TasksCount() int {
	return len(w.tasks)
}

func worker(tasksC <-chan func(), quit <-chan struct{}) {
	for {
		select {
		case <-quit:
			return
		case job := <-tasksC:
			job()
		}
	}
}
