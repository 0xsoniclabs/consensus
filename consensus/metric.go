// Copyright (c) 2026 Sonic Operations Ltd

package consensus

import (
	"fmt"
)

type Metric struct {
	Num  uint32
	Size uint64
}

func (m Metric) String() string {
	return fmt.Sprintf("{Num=%d,Size=%d}", m.Num, m.Size)
}
