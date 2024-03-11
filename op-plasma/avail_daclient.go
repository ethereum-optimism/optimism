package plasma

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-plasma/avail/types"
	"github.com/ethereum-optimism/optimism/op-plasma/avail/utils"
)

type AvailDAClient struct {
	*DAClient
}

// GetInput returns the input data corresponding to a given commitment bytes
func (c *AvailDAClient) GetInput(ctx context.Context, refKey []byte) ([]byte, error) {
	avail_blk_ref := types.AvailBlockRef{}
	err := avail_blk_ref.UnmarshalFromBinary(refKey)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to unmarshal the ethereum tx data to avail block reference, error: %v", err)
	}

	txData, err := utils.GetBlockExtrinsicData(avail_blk_ref)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to get block extrinsic data: %v", err)
	}
	return txData, nil
}

// SetInput sets the input data and returns the KZG commitment hash from Avail DA
func (c *AvailDAClient) SetInput(ctx context.Context, img []byte) ([]byte, error) {

	if len(img) >= 512000 {
		return []byte{}, ErrNotFound
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
