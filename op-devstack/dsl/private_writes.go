package dsl

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-private-interop/writes"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// PrivateWriteProbe exercises committed storage and reverted writes through a real private EL.
type PrivateWriteProbe struct {
	commonImpl
	private, projection *L2ELNode
	sender              *EOA
	address             common.Address
	expected            writes.Record
}

func NewPrivateWriteProbe(sender *EOA, private, projection *L2ELNode) *PrivateWriteProbe {
	p := &PrivateWriteProbe{commonImpl: sender.commonImpl, private: private, projection: projection, sender: sender}
	// Store calldata word 0 in slot 0; revert after the write when calldata is 64 bytes.
	initCode := common.FromHex("0x6010600c60003960106000f35f355f5536604014600c57005b5f5ffd")
	receipt := p.send(nil, initCode)
	p.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	p.address = receipt.ContractAddress
	records := p.blockWrites(receipt.BlockHash)
	p.require.Contains(tags(records), writes.Tag(bigs.Uint64Strict(private.ChainID().ToBig()), writes.Existence, p.address, common.Hash{}), "creation must publish existence")
	return p
}

func (p *PrivateWriteProbe) send(to *common.Address, data []byte) *types.Receipt {
	tx := txplan.NewPlannedTx(p.sender.Plan(), txplan.WithTo(to), txplan.WithData(data), txplan.WithGasLimit(100_000))
	receipt, err := tx.Included.Eval(p.ctx)
	p.require.NoError(err)
	return receipt
}

func (p *PrivateWriteProbe) blockWrites(hash common.Hash) []writes.Record {
	records, err := (writes.Source{RPC: p.private.inner.L2EthClient().RPC()}).FetchWrites(p.ctx, hash)
	p.require.NoError(err, "private full-block write extraction")
	return records
}

func tags(records []writes.Record) []common.Hash {
	out := make([]common.Hash, len(records))
	for i, r := range records {
		out[i] = r.Tag
	}
	return out
}

func (p *PrivateWriteProbe) Write(value uint64) {
	word := common.BigToHash(new(big.Int).SetUint64(value))
	receipt := p.send(&p.address, word[:])
	p.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	chain := bigs.Uint64Strict(p.private.ChainID().ToBig())
	tag := writes.Tag(chain, writes.Storage, p.address, common.Hash{})
	p.expected = writes.Record{Tag: tag, ValueCommitment: writes.Commit(tag, word), BlockNumber: bigs.Uint64Strict(receipt.BlockNumber)}
	records := p.blockWrites(receipt.BlockHash)
	p.require.Contains(records, p.expected, "successful storage write must be discoverable")
	p.require.Contains(tags(records), writes.Tag(chain, writes.Storage, predeploys.L1BlockAddr, common.HexToHash("0x03")), "L1 attributes deposit sequence update must be included")
	p.require.Contains(tags(records), writes.Tag(chain, writes.Nonce, p.sender.Address(), common.Hash{}), "sender nonce must be included")
	p.require.Contains(tags(records), writes.Tag(chain, writes.Balance, p.sender.Address(), common.Hash{}), "transaction fees must be included")
}

func (p *PrivateWriteProbe) RevertWrite(value uint64) {
	word := common.BigToHash(new(big.Int).SetUint64(value))
	receipt := p.send(&p.address, append(word[:], make([]byte, 32)...))
	p.require.Equal(types.ReceiptStatusFailed, receipt.Status)
	records := p.blockWrites(receipt.BlockHash)
	p.require.NotContains(tags(records), p.expected.Tag, "reverted SSTORE must not be published")
	p.require.Contains(tags(records), writes.Tag(bigs.Uint64Strict(p.private.ChainID().ToBig()), writes.Nonce, p.sender.Address(), common.Hash{}), "failed transaction still consumes nonce")
}

// VerifyPublished checks the accepted projection claim, not merely the operator's RPC output.
func (p *PrivateWriteProbe) VerifyPublished() {
	p.log.Info("Waiting for public claim to publish private storage write", "block", p.expected.BlockNumber, "tag", p.expected.Tag)
	cursor := uint64(1)
	var found *codec.RangeClaim
	var settled *optypes.Receipt
	p.require.NoError(retry.Do0(p.ctx, 120, &retry.FixedStrategy{Dur: time.Second}, func() error {
		head, err := p.projection.blockRefByLabel(eth.Unsafe)
		if err != nil {
			return err
		}
		for cursor <= head.Number {
			_, txs, err := p.projection.inner.EthClient().InfoAndTxsByNumber(p.ctx, cursor)
			if err != nil {
				return err
			}
			for _, tx := range txs {
				if tx.To() == nil || *tx.To() != predeploys.ClaimRegistryAddr || len(tx.Data()) < 4 || !bytes.Equal(tx.Data()[:4], render.PostClaimSelector[:]) {
					continue
				}
				claim, err := codec.Decode(tx.Data()[4:])
				if err != nil {
					return err
				}
				if claim.FirstBlock > p.expected.BlockNumber || claim.LastBlock < p.expected.BlockNumber {
					continue
				}
				receipt, err := p.projection.inner.EthClient().TransactionReceipt(p.ctx, tx.Hash())
				if err != nil {
					return err
				}
				found, settled = claim, receipt
				return nil
			}
			cursor++
		}
		return fmt.Errorf("no accepted write claim for private block %d", p.expected.BlockNumber)
	}))
	p.require.NotNil(found)
	p.require.NotNil(settled)
	p.require.Equal(types.ReceiptStatusSuccessful, settled.Status, "write claim must be accepted by the registry")
	p.require.Contains(found.Writes, p.expected, "batch claim must retain the exact storage value commitment and version")
	p.require.Empty(settled.Logs, "write publication must preserve projected log positions")
}
