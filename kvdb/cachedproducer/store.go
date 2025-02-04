package cachedproducer

import "github.com/0xsoniclabs/consensus/kvdb"

type StoreWithFn struct {
	kvdb.Store
	CloseFn func() error
	DropFn  func()
}

func (s *StoreWithFn) Close() error {
	return s.CloseFn()
}

func (s *StoreWithFn) Drop() {
	s.DropFn()
}
