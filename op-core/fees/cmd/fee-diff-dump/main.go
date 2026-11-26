// fee-diff-dump writes a deterministic corpus and the Go fee results for it.
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/fees"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

type feeCase struct {
	Tx             hexutil.Bytes `json:"tx"`
	Gas            uint64        `json:"gas"`
	BaseFee        uint64        `json:"base_fee"`
	BlobBaseFee    uint64        `json:"blob_base_fee"`
	BaseScalar     uint32        `json:"base_scalar"`
	BlobScalar     uint32        `json:"blob_scalar"`
	Overhead       uint64        `json:"overhead"`
	OperatorScalar uint32        `json:"operator_scalar"`
	OperatorConst  uint64        `json:"operator_const"`
	DAScalar       uint16        `json:"da_scalar"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: fee-diff-dump CORPUS_FILE")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) error {
	corpus, err := os.Create(path)
	if err != nil {
		return err
	}
	defer corpus.Close()
	encoder := json.NewEncoder(corpus)
	out := bufio.NewWriter(os.Stdout)

	source := rand.NewChaCha8([32]byte{42})
	rng := rand.New(source)
	for _, size := range []int{0, 1, 31, 32, 33, 128, 256, 1024, 8191, 8192, 8193, 16384, 32768} {
		random := make([]byte, size)
		_, _ = source.Read(random)

		// Repeat a random block so the corpus exercises actual matches, as well
		// as input that mostly consists of literals.
		for _, period := range []int{0, 64, 512, 8192} {
			data := random
			if period != 0 {
				block := random[:min(period, size)]
				data = bytes.Repeat(block, max(1, (size+period-1)/period))[:size]
			}
			for variant := range 4 {
				c := feeCase{
					Gas:            1_000_000 + rng.Uint64N(2_000_000),
					BaseFee:        rng.Uint64N(100_000_000_000) + 1,
					BlobBaseFee:    rng.Uint64N(100_000_000_000) + 1,
					BaseScalar:     rng.Uint32N(10_000_000) + 1,
					BlobScalar:     rng.Uint32N(10_000_000) + 1,
					Overhead:       rng.Uint64N(10_000),
					OperatorScalar: rng.Uint32N(10_000_000),
					OperatorConst:  rng.Uint64N(100_000_000),
					DAScalar:       uint16(rng.Uint32N(65_535) + 1),
				}
				if variant == 0 {
					c.OperatorScalar, c.OperatorConst, c.DAScalar = 0, 0, 0
				}
				to := common.Address{19: 1}
				tx := types.NewTx(&types.DynamicFeeTx{
					ChainID: big.NewInt(10), Gas: c.Gas, To: &to, Data: data,
					GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1),
					R: big.NewInt(1), S: big.NewInt(1),
				})
				c.Tx, err = tx.MarshalBinary()
				if err != nil {
					return err
				}
				if err := encoder.Encode(c); err != nil {
					return err
				}
				cost := fees.NewRollupCostData(c.Tx)
				baseFee := new(big.Int).SetUint64(c.BaseFee)
				baseScalar := new(big.Int).SetUint64(uint64(c.BaseScalar))
				params := fees.L1FeeParams{
					L1BaseFee: baseFee, BaseFeeScalar: baseScalar,
					L1BlobBaseFee: new(big.Int).SetUint64(c.BlobBaseFee),
					BlobFeeScalar: new(big.Int).SetUint64(uint64(c.BlobScalar)),
				}
				bedrock := fees.L1CostBedrock(cost.Zeroes*4+(cost.Ones+68)*16, baseFee, new(big.Int).SetUint64(c.Overhead), baseScalar)
				regolith := fees.L1CostBedrock(cost.Zeroes*4+cost.Ones*16, baseFee, new(big.Int).SetUint64(c.Overhead), baseScalar)
				ecotone := fees.L1CostEcotone(cost, params)
				fjord := fees.L1CostFjord(cost, params)
				isthmus := fees.OperatorCostIsthmus(c.Gas, uint64(c.OperatorScalar), c.OperatorConst)
				jovian := fees.OperatorCostJovian(c.Gas, uint64(c.OperatorScalar), c.OperatorConst)
				footprint, err := daFootprint(c)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintln(out, bedrock, regolith, ecotone, fjord, isthmus, jovian, footprint); err != nil {
					return err
				}
			}
		}
	}
	return out.Flush()
}

func daFootprint(c feeCase) (uint64, error) {
	// Put one user transaction after the L1 attributes deposit so the block
	// helper's result is the footprint of that transaction.
	data := make([]byte, fees.JovianL1AttributesLen)
	binary.BigEndian.PutUint32(data, 0x3db6be2b) // setL1BlockValuesJovian()
	binary.BigEndian.PutUint16(data[len(data)-2:], c.DAScalar)
	to := common.HexToAddress("0x4200000000000000000000000000000000000015")
	deposit, err := (&optypes.DepositTx{To: &to, Gas: 1_000_000, Data: data}).MarshalBinary()
	if err != nil {
		return 0, err
	}
	return fees.CalcDAFootprint([]hexutil.Bytes{deposit, c.Tx})
}
