package monitor

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// fakeChain is an in-memory L2 chain for the expiry checks. Its blocks have fixed times, and its
// storage and balances change at given blocks.
type fakeChain struct {
	id        eth.ChainID
	headers   []*types.Header
	receipts  map[uint64][]*types.Receipt
	storage   map[[2]common.Hash][]change
	balances  map[common.Address][]change
	safe      uint64
	blockTime uint64
}

type change struct {
	block uint64
	value *big.Int
}

const fakeGenesisTime = 1000

func newFakeChain(id uint64, blocks uint64, blockTime uint64) *fakeChain {
	c := &fakeChain{
		id:        eth.ChainIDFromUInt64(id),
		receipts:  make(map[uint64][]*types.Receipt),
		storage:   make(map[[2]common.Hash][]change),
		balances:  make(map[common.Address][]change),
		blockTime: blockTime,
	}
	for i := uint64(0); i < blocks; i++ {
		h := &types.Header{Number: new(big.Int).SetUint64(i), Time: fakeGenesisTime + i*blockTime,
			Extra: big.NewInt(int64(id)).Bytes()}
		if i > 0 {
			h.ParentHash = c.headers[i-1].Hash()
		}
		c.headers = append(c.headers, h)
	}
	c.safe = blocks - 1
	return c
}

func (c *fakeChain) blockAt(ts uint64) uint64 { return (ts - fakeGenesisTime) / c.blockTime }
func (c *fakeChain) hashOf(n uint64) common.Hash {
	return c.headers[n].Hash()
}

// addLog adds a log to the receipt of tx `tx` in block `n`, creating the receipt if needed.
func (c *fakeChain) addLog(n uint64, tx common.Hash, l *types.Log) *types.Log {
	var rcpt *types.Receipt
	for _, r := range c.receipts[n] {
		if r.TxHash == tx {
			rcpt = r
		}
	}
	if rcpt == nil {
		rcpt = &types.Receipt{TxHash: tx, BlockNumber: new(big.Int).SetUint64(n), BlockHash: c.hashOf(n),
			Status: types.ReceiptStatusSuccessful}
		c.receipts[n] = append(c.receipts[n], rcpt)
	}
	index := uint(0)
	for _, r := range c.receipts[n] {
		index += uint(len(r.Logs))
	}
	l.BlockNumber, l.BlockHash, l.TxHash, l.Index = n, c.hashOf(n), tx, index
	rcpt.Logs = append(rcpt.Logs, l)
	return l
}

// removeLogs drops every log with the given topic.
func (c *fakeChain) removeLogs(topic common.Hash) {
	for _, rs := range c.receipts {
		for _, r := range rs {
			r.Logs = slices.DeleteFunc(r.Logs, func(l *types.Log) bool { return l.Topics[0] == topic })
		}
	}
}

func (c *fakeChain) setStorage(addr common.Address, slot common.Hash, n uint64, value *big.Int) {
	key := [2]common.Hash{common.BytesToHash(addr.Bytes()), slot}
	c.storage[key] = append(c.storage[key], change{block: n, value: value})
}

func (c *fakeChain) setBalance(addr common.Address, n uint64, value *big.Int) {
	c.balances[addr] = append(c.balances[addr], change{block: n, value: value})
}

func valueAt(changes []change, n uint64) *big.Int {
	v := new(big.Int)
	for _, ch := range changes {
		if ch.block <= n {
			v = ch.value
		}
	}
	return v
}

func (c *fakeChain) info(n uint64) (eth.BlockInfo, error) {
	if n >= uint64(len(c.headers)) {
		return nil, ethereum.NotFound
	}
	return eth.HeaderBlockInfo(c.headers[n]), nil
}

func (c *fakeChain) InfoByNumber(_ context.Context, n uint64) (eth.BlockInfo, error) {
	return c.info(n)
}

// InfoByLabel serves `safe` as the block of every label the checks read up to.
func (c *fakeChain) InfoByLabel(_ context.Context, _ eth.BlockLabel) (eth.BlockInfo, error) {
	return c.info(c.safe)
}

func (c *fakeChain) FetchReceiptsByNumber(_ context.Context, n uint64) (eth.BlockInfo, optypes.Receipts, error) {
	info, err := c.info(n)
	if err != nil {
		return nil, nil, err
	}
	return info, optypes.FromGethReceipts(c.receipts[n]), nil
}

func (c *fakeChain) GetStorageAt(_ context.Context, addr common.Address, slot common.Hash, blockTag string) (common.Hash, error) {
	n, err := strconv.ParseUint(strings.TrimPrefix(blockTag, "0x"), 16, 64)
	if err != nil {
		return common.Hash{}, err
	}
	if n > c.safe {
		return common.Hash{}, errors.New("storage read above the safe head")
	}
	return common.BigToHash(valueAt(c.storage[[2]common.Hash{common.BytesToHash(addr.Bytes()), slot}], n)), nil
}

func (c *fakeChain) BalanceAt(_ context.Context, addr common.Address, n *big.Int) (*big.Int, error) {
	return valueAt(c.balances[addr], bigs.Uint64Strict(n)), nil
}

func (c *fakeChain) FilterLogs(_ context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	from, to := bigs.Uint64Strict(q.FromBlock), bigs.Uint64Strict(q.ToBlock)
	if to-from >= maxLogRange {
		return nil, errors.New("log range too large")
	}
	var out []types.Log
	for n := from; n <= to && n < uint64(len(c.headers)); n++ {
		for _, r := range c.receipts[n] {
			for _, l := range r.Logs {
				if logMatches(l, q) {
					out = append(out, *l)
				}
			}
		}
	}
	return out, nil
}

func logMatches(l *types.Log, q ethereum.FilterQuery) bool {
	if len(q.Addresses) > 0 && !slices.Contains(q.Addresses, l.Address) {
		return false
	}
	for i, options := range q.Topics {
		if len(options) == 0 {
			continue
		}
		if i >= len(l.Topics) || !slices.Contains(options, l.Topics[i]) {
			return false
		}
	}
	return true
}

var _ ExpiryClient = (*fakeChain)(nil)

func addrTopic(a common.Address) common.Hash { return common.BytesToHash(a.Bytes()) }
func word(v *big.Int) []byte                 { return common.BigToHash(v).Bytes() }

func sentMessageLog(m *l2ToL2Message) *types.Log {
	data, err := sentMessageDataArgs.Pack(m.Sender, m.Message)
	if err != nil {
		panic(err)
	}
	return &types.Log{Address: predeploys.L2toL2CrossDomainMessengerAddr,
		Topics: []common.Hash{sentMessageTopic, common.BigToHash(m.Destination.ToBig()), addrTopic(m.Target), common.BigToHash(m.Nonce)},
		Data:   data}
}

func relayETHCalldata(from, to common.Address, amount *big.Int) []byte {
	args, err := relayETHArgs.Pack(from, to, amount)
	if err != nil {
		panic(err)
	}
	return append(slices.Clone(relayETHSelector), args...)
}

func sendETHLog(from, to common.Address, amount *big.Int, dest eth.ChainID) *types.Log {
	return &types.Log{Address: predeploys.SuperchainETHBridgeAddr,
		Topics: []common.Hash{sendETHTopic, addrTopic(from), addrTopic(to)},
		Data:   append(word(amount), word(dest.ToBig())...)}
}

func liquidityLog(topic common.Hash, who common.Address, amount *big.Int) *types.Log {
	return &types.Log{Address: predeploys.ETHLiquidityAddr, Topics: []common.Hash{topic, addrTopic(who)}, Data: word(amount)}
}

func messageExpiredLog(hash common.Hash, t uint64) *types.Log {
	return &types.Log{Address: predeploys.L2toL2CrossDomainMessengerAddr,
		Topics: []common.Hash{messageExpiredTopic, hash}, Data: word(new(big.Int).SetUint64(t))}
}

func exportLog(hash common.Hash, source eth.ChainID, messenger common.Address, t uint64) *types.Log {
	return &types.Log{Address: predeploys.UndeliveredMessageExporterAddr,
		Topics: []common.Hash{undeliveredExpTopic, hash, common.BigToHash(source.ToBig())},
		Data:   append(word(new(big.Int).SetBytes(messenger.Bytes())), word(new(big.Int).SetUint64(t))...)}
}

func relayedMessageLog(source eth.ChainID, nonce *big.Int, hash common.Hash) *types.Log {
	return &types.Log{Address: predeploys.L2toL2CrossDomainMessengerAddr,
		Topics: []common.Hash{relayedMessageTopic, common.BigToHash(source.ToBig()), common.BigToHash(nonce), hash},
		Data:   make([]byte, 32)}
}

func refundETHLog(from common.Address, amount *big.Int, hash common.Hash) *types.Log {
	return &types.Log{Address: predeploys.SuperchainETHBridgeAddr,
		Topics: []common.Hash{refundETHTopic, addrTopic(from), hash}, Data: word(amount)}
}

func txHash(name string) common.Hash { return crypto.Keccak256Hash([]byte(name)) }

// expiryScenario is a valid expiry flow: A sends ETH to B, B never relays it, B exports the fact
// once the expiry period has passed, A marks the message expired, and the sender is refunded.
type expiryScenario struct {
	a, b    *fakeChain
	msg     *l2ToL2Message
	hash    common.Hash
	sender  common.Address
	to      common.Address
	amount  *big.Int
	sentAt  uint64
	exportT uint64

	sendBlock, exportBlock, expireBlock, refundBlock uint64

	expiredLog, refundLog *types.Log
}

const (
	scenarioWindow = 12
	scenarioPeriod = 30
)

var liquidityGenesisBalance = new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))

func newExpiryScenario() *expiryScenario {
	a, b := newFakeChain(901, 80, 2), newFakeChain(902, 80, 2)
	s := &expiryScenario{
		a: a, b: b,
		sender:    common.HexToAddress("0x1111111111111111111111111111111111111111"),
		to:        common.HexToAddress("0x2222222222222222222222222222222222222222"),
		amount:    new(big.Int).Div(big.NewInt(params.Ether), big.NewInt(2)),
		sendBlock: 10, expireBlock: 40, refundBlock: 45,
	}
	s.sentAt = a.headers[s.sendBlock].Time
	s.exportT = s.sentAt + scenarioPeriod + 2
	s.exportBlock = b.blockAt(s.exportT)
	s.msg = &l2ToL2Message{
		Source: a.id, Destination: b.id, Nonce: big.NewInt(3),
		Sender: predeploys.SuperchainETHBridgeAddr, Target: predeploys.SuperchainETHBridgeAddr,
		Message: relayETHCalldata(s.sender, s.to, s.amount),
	}
	s.hash = s.msg.Hash()

	for _, c := range []*fakeChain{a, b} {
		c.setStorage(predeploys.L2toL2CrossDomainMessengerAddr, common.BigToHash(big.NewInt(expiryPeriodSlot)), 0, big.NewInt(scenarioPeriod))
		c.setBalance(predeploys.ETHLiquidityAddr, 0, liquidityGenesisBalance)
	}

	// The send on A: burn, SentMessage, SendETH.
	sendTx := txHash("send")
	a.addLog(s.sendBlock, sendTx, liquidityLog(liquidityBurnedTopic, predeploys.SuperchainETHBridgeAddr, s.amount))
	a.addLog(s.sendBlock, sendTx, sentMessageLog(s.msg))
	a.addLog(s.sendBlock, sendTx, sendETHLog(s.sender, s.to, s.amount, b.id))
	a.setStorage(predeploys.L2toL2CrossDomainMessengerAddr, mappingSlot(s.hash, sentMessageTimestampsSlot), s.sendBlock, new(big.Int).SetUint64(s.sentAt))
	a.setBalance(predeploys.ETHLiquidityAddr, s.sendBlock, new(big.Int).Add(liquidityGenesisBalance, s.amount))

	// Another message in the same block, which must not be mistaken for it.
	other := *s.msg
	other.Nonce = big.NewInt(4)
	a.addLog(s.sendBlock, txHash("other send"), sentMessageLog(&other))

	// B exports the fact once the expiry period has passed.
	b.addLog(s.exportBlock, txHash("export"), exportLog(s.hash, a.id, common.HexToAddress("0xa1"), s.exportT))

	// A marks the message expired.
	s.expiredLog = a.addLog(s.expireBlock, txHash("expire"), messageExpiredLog(s.hash, s.exportT))
	a.setStorage(predeploys.L2toL2CrossDomainMessengerAddr, mappingSlot(s.hash, expiredMessagesSlot), s.expireBlock, big.NewInt(1))

	// A refunds the sender: mint, RefundETH.
	a.addLog(s.refundBlock, txHash("refund"), liquidityLog(liquidityMintedTopic, predeploys.SuperchainETHBridgeAddr, s.amount))
	s.refundLog = a.addLog(s.refundBlock, txHash("refund"), refundETHLog(s.sender, s.amount, s.hash))
	a.setBalance(predeploys.ETHLiquidityAddr, s.refundBlock, liquidityGenesisBalance)
	return s
}

func (s *expiryScenario) verifier() *expiryVerifier {
	return &expiryVerifier{
		clients:             map[eth.ChainID]ExpiryClient{s.a.id: s.a, s.b.id: s.b},
		messageExpiryWindow: scenarioWindow,
		head:                eth.Finalized,
	}
}

func relayETHLog(from, to common.Address, amount *big.Int, source eth.ChainID) *types.Log {
	return &types.Log{Address: predeploys.SuperchainETHBridgeAddr,
		Topics: []common.Hash{relayETHTopic, addrTopic(from), addrTopic(to)},
		Data:   append(word(amount), word(source.ToBig())...)}
}
