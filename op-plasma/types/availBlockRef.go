package types

import (
	"encoding/json"
	"fmt"
)

type AvailBlockRef struct {
	BlockHash string // Hash for block on avail chain
	Sender    string // sender address to filter extrinsic out sepecifically for this address
	Nonce     int64  // nonce to filter specific extrinsic
}

func (a *AvailBlockRef) MarshalToBinary() ([]byte, error) {
	ref_bytes, err := json.Marshal(a)
	if err != nil {
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%v", err)
	}
	return ref_bytes, nil
}

func (a *AvailBlockRef) UnmarshalFromBinary(avail_Blk_Ref []byte) error {
	err := json.Unmarshal(avail_Blk_Ref, a)
	if err != nil {
		return fmt.Errorf("unable to convert avail_Blk_Ref bytes to AvailBlockRef Struct and getting error:%v", err)
	}
	return nil
}
