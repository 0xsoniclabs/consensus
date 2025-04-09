// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusengine

type Config struct {
	// Suppresses the frame missmatch panic - used only for importing older historical event files, disabled by default
	SuppressFramePanic bool
}

// DefaultConfig for livenet.
func DefaultConfig() Config {
	return Config{
		SuppressFramePanic: false,
	}
}
