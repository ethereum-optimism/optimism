package plasma

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-plasma/avail/utils"
)

type AvailDAClient struct {
	*DAClient
}

// GetInput returns the input data corresponding to a given commitment bytes
func (c *AvailDAClient) GetInput(ctx context.Context, key []byte) ([]byte, error) {
	return nil, nil
}

// SetInput sets the input data and returns the KZG commitment hash from Avail DA
func (c *AvailDAClient) SetInput(ctx context.Context, img []byte) ([]byte, error) {
	if len(img) >= 512000 {
		return []byte{}, fmt.Errorf("size of TxData is more than 512KB, it is higher than a single data submit transaction supports on avail")
	}

	avail_Blk_Ref, err := utils.SubmitDataAndWatch(ctx, img)

	if err != nil {
		return []byte{}, fmt.Errorf("cannot submit data:%v", err)
	}

	ref_bytes_data, err := avail_Blk_Ref.MarshalToBinary()

	if err != nil {
		return []byte{}, fmt.Errorf("cannot get the binary form of avail block reference:%v", err)
	}
	return ref_bytes_data, nil
}
