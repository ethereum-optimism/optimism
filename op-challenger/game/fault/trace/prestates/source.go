package prestates

import (
	"net/url"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
)

func NewPrestateSource(baseURL *url.URL, path string, localDir string, stateConverter vm.StateConverter) PrestateSource {
	if baseURL != nil {
		return NewMultiPrestateProvider(baseURL, localDir, stateConverter)
	} else {
		return NewSinglePrestateSource(path)
	}
}
