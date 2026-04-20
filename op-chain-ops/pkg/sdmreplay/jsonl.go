package sdmreplay

import (
	"encoding/json"
	"io"
)

// WriteJSONL emits records in stable order: run config, tx rows, block rows, mismatch rows, summary.
func WriteJSONL(w io.Writer, result *RangeResult, summaryOnly bool) error {
	enc := json.NewEncoder(w)
	if err := enc.Encode(result.RunConfig); err != nil {
		return err
	}
	for _, block := range result.Blocks {
		if !summaryOnly {
			for _, tx := range block.Txs {
				if err := enc.Encode(tx); err != nil {
					return err
				}
			}
		}
		if err := enc.Encode(block.Block); err != nil {
			return err
		}
		for _, mismatch := range block.Mismatches {
			if err := enc.Encode(mismatch); err != nil {
				return err
			}
		}
	}
	return enc.Encode(result.Summary)
}
