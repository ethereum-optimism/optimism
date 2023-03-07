package reassemble

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sort"

	"github.com/ethereum-optimism/optimism/op-node/cmd/batch_decoder/fetch"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
)

type ChannelWithMeta struct {
	ID            derive.ChannelID    `json:"id"`
	SkippedFrames []FrameWithMetadata `json:"skipped_frames"`
	IsReady       bool                `json:"is_ready"`
	Frames        []FrameWithMetadata `json:"frames"`
}

type FrameWithMetadata struct {
	TxHash         common.Hash  `json:"transaction_hash"`
	InclusionBlock uint64       `json:"inclusion_block"`
	Frame          derive.Frame `json:"frame"`
}

type Config struct {
	BatchInbox   common.Address
	InDirectory  string
	OutDirectory string
}

// Channels loads all transactions from the given input directory that are submitted to the
// specified batch inbox and then re-assembles all channels & writes the re-assembled channels
// to the out directory.
func Channels(config Config) {
	if err := os.MkdirAll(config.OutDirectory, 0750); err != nil {
		log.Fatal(err)
	}
	txns := loadTransactions(config.InDirectory, config.BatchInbox)
	// Sort first by block number then by transaction index inside the block number range.
	// This is to match the order they are processed in derivation.
	sort.Slice(txns, func(i, j int) bool {
		if txns[i].BlockNumber == txns[j].BlockNumber {
			return txns[i].TxIndex < txns[j].TxIndex
		} else {
			return txns[i].BlockNumber < txns[j].BlockNumber
		}

	})
	frames := transactionsToFrames(txns)
	framesByChannel := make(map[derive.ChannelID][]FrameWithMetadata)
	for _, frame := range frames {
		framesByChannel[frame.Frame.ID] = append(framesByChannel[frame.Frame.ID], frame)
	}
	for id, frames := range framesByChannel {
		ch := processFrames(id, frames)
		filename := path.Join(config.OutDirectory, fmt.Sprintf("%s.json", id.String()))
		file, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		enc := json.NewEncoder(file)
		if err := enc.Encode(ch); err != nil {
			log.Fatal(err)
		}
	}
}

func processFrames(id derive.ChannelID, frames []FrameWithMetadata) ChannelWithMeta {
	// TO DO: Use the same approach as in derivation.
	ready := false
	for _, frame := range frames {
		ready = frame.Frame.IsLast || ready
	}
	if !ready {
		fmt.Printf("Found channel that was not closed: %v\n", id.String())
	}
	return ChannelWithMeta{
		ID:            id,
		Frames:        frames,
		SkippedFrames: nil,
		IsReady:       ready,
	}
}

func transactionsToFrames(txns []fetch.TransactionWithMeta) []FrameWithMetadata {
	var out []FrameWithMetadata
	for _, tx := range txns {
		for _, frame := range tx.Frames {
			fm := FrameWithMetadata{
				TxHash:         tx.Tx.Hash(),
				InclusionBlock: tx.BlockNumber,
				Frame:          frame,
			}
			out = append(out, fm)
		}
	}
	return out
}

func loadTransactions(dir string, inbox common.Address) []fetch.TransactionWithMeta {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	var out []fetch.TransactionWithMeta
	for _, file := range files {
		f, err := os.Open(path.Join(dir, file.Name()))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		var txm fetch.TransactionWithMeta
		if err := dec.Decode(&txm); err != nil {
			log.Fatalf("Failed to decode %v. Err: %v\n", path.Join(dir, file.Name()), err)
		}
		if txm.InboxAddr == inbox && txm.ValidSender {
			out = append(out, txm)
		}
	}
	return out
}
