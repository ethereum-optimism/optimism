package bindings

import (
	"strings"

	"github.com/ledgerwatch/erigon/accounts/abi"
)

type MetaData struct {
	ABI string
	Bin string
}

func (m *MetaData) GetAbi() (*abi.ABI, error) {
	abi, err := abi.JSON(strings.NewReader(m.ABI))
	if err != nil {
		return nil, err
	}
	return &abi, nil
}
