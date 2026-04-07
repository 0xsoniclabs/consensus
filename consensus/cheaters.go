// Copyright (c) 2026 Sonic Operations Ltd

package consensus

// Cheaters is a slice type for storing cheaters list.
type Cheaters []ValidatorID

// Set returns map of cheaters
func (s Cheaters) Set() map[ValidatorID]struct{} {
	set := map[ValidatorID]struct{}{}
	for _, element := range s {
		set[element] = struct{}{}
	}
	return set
}
