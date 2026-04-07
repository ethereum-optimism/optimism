package filter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BlockSelector selects either the latest block or a block at a specific height.
// It accepts the JSON string "latest", a JSON number, or a quoted numeric string.
type BlockSelector struct {
	latest bool
	number uint64
}

func LatestBlockSelector() BlockSelector {
	return BlockSelector{latest: true}
}

func BlockSelectorFromNumber(number uint64) BlockSelector {
	return BlockSelector{number: number}
}

func (s BlockSelector) Latest() bool {
	return s.latest
}

func (s BlockSelector) Number() uint64 {
	return s.number
}

func (s *BlockSelector) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if input == "" {
		return fmt.Errorf("empty block selector")
	}

	if input == `"latest"` {
		*s = LatestBlockSelector()
		return nil
	}

	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		number, err := parseQuotedBlockNumber(input[1 : len(input)-1])
		if err != nil {
			return err
		}
		*s = BlockSelectorFromNumber(number)
		return nil
	}

	var number uint64
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("invalid block selector %q: %w", input, err)
	}
	*s = BlockSelectorFromNumber(number)
	return nil
}

func parseQuotedBlockNumber(input string) (uint64, error) {
	switch input {
	case "":
		return 0, fmt.Errorf("empty block selector")
	}

	if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
		number, err := hexutil.DecodeUint64(input)
		if err != nil {
			return 0, fmt.Errorf("invalid hex block selector %q: %w", input, err)
		}
		return number, nil
	}

	number, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid block selector %q: %w", input, err)
	}
	return number, nil
}
