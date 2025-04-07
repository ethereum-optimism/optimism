package chainconfig

import (
	"fmt"
)

// GetSepoliaIsthmusTime returns the Isthmus hardfork time for the Sepolia network
func GetSepoliaIsthmusTime() (uint64, error) {
	config, err := LoadSepoliaIsthmusConfig()
	if err != nil {
		return 0, fmt.Errorf("failed to load Sepolia Isthmus config: %w", err)
	}
	return config.IsthmusTime, nil
}
