package avail

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-plasma/cmd/avail/scripts"
	"github.com/ethereum-optimism/optimism/op-plasma/cmd/avail/types"
)

type AvailService struct {
	Seed    string              `json:"seed"`
	ApiURL  string              `json:"api_url"`
	AppID   int                 `json:"app_id"`
	Timeout time.Duration       `json:"timeout"`
	Specs   *types.AvailDASpecs `json:"availDASpecs"`
}

func NewAvailService(apiURL string, seed string, appID int, timeout time.Duration) *AvailService {

	availSpecs, err := types.NewAvailDASpecs(apiURL, appID, seed, timeout)

	if err != nil {
		panic("failed avail initialisation")
	}

	return &AvailService{
		Seed:    seed,
		ApiURL:  apiURL,
		AppID:   appID,
		Timeout: timeout,
		Specs:   availSpecs,
	}
}

func (s *AvailService) Get(ctx context.Context, comm []byte) ([]byte, error) {
	avail_blk_ref := types.AvailBlockRef{}
	err := avail_blk_ref.UnmarshalFromBinary(comm[1:])
	if err != nil {
		return []byte{}, fmt.Errorf("failed to unmarshal the ethereum tx data to avail block reference, error: %w", err)
	}

	input, err := scripts.GetBlockExtrinsicData(*s.Specs, avail_blk_ref)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to get block extrinsic data: %w", err)
	}

	return input, nil
}

func (s *AvailService) Put(ctx context.Context, value []byte) ([]byte, error) {

	if len(value) >= 512000 {
		return nil, fmt.Errorf("the length of input cannot be greater than 512kb")
	}

	avail_Blk_Ref, err := scripts.SubmitDataAndWatch(s.Specs, ctx, value)

	if err != nil {
		return nil, fmt.Errorf("cannot submit data:%w", err)
	}

	comm, err := avail_Blk_Ref.MarshalToBinary()

	if err != nil {
		return nil, fmt.Errorf("cannot get the binary form of avail block reference:%w", err)
	}

	return comm, nil
}
