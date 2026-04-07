// Copyright (c) 2026 Sonic Operations Ltd

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
