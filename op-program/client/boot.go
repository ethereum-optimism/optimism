package client

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type BootInfo struct {
	// TODO(CLI-XXX): The rollup config will be hardcoded. It's configurable for testing purposes.
	Rollup             *rollup.Config      `json:"rollup"`
	L2ChainConfig      *params.ChainConfig `json:"l2_chain_config"`
	L1Head             common.Hash         `json:"l1_head"`
	L2Head             common.Hash         `json:"l2_head"`
	L2Claim            common.Hash         `json:"l2_claim"`
	L2ClaimBlockNumber uint64              `json:"l2_claim_block_number"`
}

type BootstrapOracleWriter struct {
	w io.Writer
}

func NewBootstrapOracleWriter(w io.Writer) *BootstrapOracleWriter {
	return &BootstrapOracleWriter{w: w}
}

func (bw *BootstrapOracleWriter) WriteBootInfo(info *BootInfo) error {
	// TODO(CLI-3751): Bootstrap from local oracle
	payload, err := json.Marshal(info)
	if err != nil {
		return err
	}
	var b []byte
	b = binary.BigEndian.AppendUint32(b, uint32(len(payload)))
	b = append(b, payload...)
	_, err = bw.w.Write(b)
	return err
}

type BootstrapOracleReader struct {
	r io.Reader
}

func NewBootstrapOracleReader(r io.Reader) *BootstrapOracleReader {
	return &BootstrapOracleReader{r: r}
}

func (br *BootstrapOracleReader) BootInfo() (*BootInfo, error) {
	var length uint32
	if err := binary.Read(br.r, binary.BigEndian, &length); err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("failed to read bootinfo length prefix: %w", err)
	}
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(br.r, payload); err != nil {
			return nil, fmt.Errorf("failed to read bootinfo data (length %d): %w", length, err)
		}
	}
	var bootInfo BootInfo
	if err := json.Unmarshal(payload, &bootInfo); err != nil {
		return nil, err
	}
	return &bootInfo, nil
}
