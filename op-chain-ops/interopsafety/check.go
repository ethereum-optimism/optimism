package interopsafety

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// depositTxType is the EIP-2718 type of an OP Stack deposit transaction.
const depositTxType = 0x7e

// logsWindow bounds one eth_getLogs range query.
const logsWindow = 2000

// Supernode is one supernode's view of every chain it follows, keyed by chain ID.
type Supernode struct {
	Name   string
	Chains map[uint64]ChainReader
}

// Failure is one safety violation or inconsistency.
type Failure struct {
	Supernode string `json:"supernode"`
	Chain     uint64 `json:"chain"`
	Block     uint64 `json:"block"`
	Reason    string `json:"reason"`
}

// ChainState is the progress of checking one chain on one supernode.
type ChainState struct {
	CrossSafe   uint64 `json:"cross_safe"`
	LocalSafe   uint64 `json:"local_safe"`
	Unsafe      uint64 `json:"unsafe"`
	CheckedTo   uint64 `json:"checked_to"`
	ExecMsgs    int    `json:"exec_msgs"`
	LastAdvance int64  `json:"last_advance"`
}

// Config sets what a Checker checks.
type Config struct {
	// From is the first block to check per chain, normally the Lagoon activation block.
	From map[uint64]uint64
	// GoneTxs are invalidated transactions that must not be canonical on any chain.
	GoneTxs []common.Hash
	// DepositOnly are replaced blocks (chain ID -> block numbers) that must hold only deposits.
	DepositOnly map[uint64][]uint64
	// StallAfter flags a supernode whose cross-safe head has not advanced for this long.
	StallAfter time.Duration
}

// Checker checks a set of supernodes incrementally: each Pass validates only the
// blocks that became cross-safe since the previous pass, and re-checks from a lower
// head if cross-safe moves back.
type Checker struct {
	cfg        Config
	log        log.Logger
	now        func() time.Time
	supernodes []*Supernode
	chains     []uint64
	state      map[string]map[uint64]*ChainState
	failures   []Failure
}

// NewChecker returns a checker over the given supernodes. Every supernode must have a
// reader for every chain in chains.
func NewChecker(lgr log.Logger, cfg Config, chains []uint64, supernodes []*Supernode, now func() time.Time) *Checker {
	state := map[string]map[uint64]*ChainState{}
	for _, sn := range supernodes {
		state[sn.Name] = map[uint64]*ChainState{}
		for _, c := range chains {
			state[sn.Name][c] = &ChainState{LastAdvance: now().Unix()}
		}
	}
	return &Checker{cfg: cfg, log: lgr, now: now, supernodes: supernodes, chains: chains, state: state}
}

// Failures returns every failure found so far.
func (k *Checker) Failures() []Failure { return k.failures }

// State returns the per-supernode, per-chain progress.
func (k *Checker) State() map[string]map[uint64]*ChainState { return k.state }

func (k *Checker) fail(sn string, chain, block uint64, err error) {
	f := Failure{Supernode: sn, Chain: chain, Block: block, Reason: err.Error()}
	k.failures = append(k.failures, f)
	k.log.Error("Safety check failed", "supernode", sn, "chain", chain, "block", block, "reason", f.Reason)
}

// ValidateMessage re-checks one executing message found in block execBlock (at time
// execTime) on chain execChain. readers holds the same supernode's view of every chain
// in the dependency set; crossSafe its cross-safe head per chain. It returns nil if the
// message is valid.
func ValidateMessage(ctx context.Context, readers map[uint64]ChainReader, crossSafe map[uint64]uint64, execTime uint64, msg *messages.Message) error {
	id := msg.Identifier
	srcBig := id.ChainID.ToBig()
	if !srcBig.IsUint64() {
		return fmt.Errorf("message from chain %s, which is not in the dependency set", id.ChainID)
	}
	src := bigs.Uint64Strict(srcBig)
	srcReader, ok := readers[src]
	if !ok {
		return fmt.Errorf("message from chain %d, which is not in the dependency set", src)
	}
	if id.Timestamp > execTime {
		return fmt.Errorf("initiating timestamp %d is after executing block timestamp %d", id.Timestamp, execTime)
	}
	if head, ok := crossSafe[src]; ok && id.BlockNumber > head && id.Timestamp < execTime {
		return fmt.Errorf("initiating block %d on chain %d is beyond the cross-safe head %d", id.BlockNumber, src, head)
	}
	_, srcTime, err := srcReader.Header(ctx, id.BlockNumber)
	if err != nil {
		return fmt.Errorf("initiating block %d on chain %d not readable: %w", id.BlockNumber, src, err)
	}
	if srcTime != id.Timestamp {
		return fmt.Errorf("identifier timestamp %d does not match block %d timestamp %d on chain %d", id.Timestamp, id.BlockNumber, srcTime, src)
	}
	logs, err := srcReader.BlockLogs(ctx, id.BlockNumber)
	if err != nil {
		return fmt.Errorf("logs of block %d on chain %d not readable: %w", id.BlockNumber, src, err)
	}
	for _, l := range logs {
		if uint32(l.Index) != id.LogIndex {
			continue
		}
		if l.Address != id.Origin {
			return fmt.Errorf("log %d in block %d on chain %d is from %s, identifier says %s", id.LogIndex, id.BlockNumber, src, l.Address, id.Origin)
		}
		if h := crypto.Keccak256Hash(messages.LogToMessagePayload(l)); h != msg.PayloadHash {
			return fmt.Errorf("payload hash of log %d in block %d on chain %d is %s, message says %s", id.LogIndex, id.BlockNumber, src, h, msg.PayloadHash)
		}
		return nil
	}
	return fmt.Errorf("no log %d in block %d on chain %d", id.LogIndex, id.BlockNumber, src)
}

// Pass reads every supernode's cross-safe heads, then validates the executing
// messages in each newly cross-safe range.
func (k *Checker) Pass(ctx context.Context) {
	now := k.now().Unix()
	for _, sn := range k.supernodes {
		for _, c := range k.chains {
			k.readHeads(ctx, sn, c, now)
		}
	}
	for _, sn := range k.supernodes {
		crossSafe := map[uint64]uint64{}
		for _, c := range k.chains {
			crossSafe[c] = k.state[sn.Name][c].CrossSafe
		}
		for _, c := range k.chains {
			k.scan(ctx, sn, c, crossSafe)
		}
	}
}

func (k *Checker) readHeads(ctx context.Context, sn *Supernode, c uint64, now int64) {
	st := k.state[sn.Name][c]
	ss, err := sn.Chains[c].SyncStatus(ctx)
	if err != nil {
		k.log.Warn("Sync status unavailable", "supernode", sn.Name, "chain", c, "err", err)
		return
	}
	switch {
	case ss.SafeL2.Number > st.CrossSafe:
		st.LastAdvance = now
	case ss.SafeL2.Number < st.CrossSafe:
		k.log.Info("Cross-safe moved back", "supernode", sn.Name, "chain", c, "from", st.CrossSafe, "to", ss.SafeL2.Number)
		st.CheckedTo = min(st.CheckedTo, ss.SafeL2.Number)
	}
	// The EL must hold the supernode's cross-safe block; parent links then pin every
	// ancestor the scan walks.
	if hash, _, err := sn.Chains[c].Header(ctx, ss.SafeL2.Number); err == nil && hash != ss.SafeL2.Hash {
		k.log.Warn("EL does not hold the cross-safe block yet; skipping this pass", "supernode", sn.Name, "chain", c, "block", ss.SafeL2.Number)
		return
	}
	st.CrossSafe, st.LocalSafe, st.Unsafe = ss.SafeL2.Number, ss.LocalSafeL2.Number, ss.UnsafeL2.Number
}

func (k *Checker) scan(ctx context.Context, sn *Supernode, c uint64, crossSafe map[uint64]uint64) {
	st := k.state[sn.Name][c]
	for lo := max(st.CheckedTo+1, k.cfg.From[c]); lo <= st.CrossSafe; lo += logsWindow {
		hi := min(lo+logsWindow-1, st.CrossSafe)
		logs, err := sn.Chains[c].ExecutingMessageLogs(ctx, lo, hi)
		if err != nil {
			k.log.Warn("Executing message logs unavailable", "supernode", sn.Name, "chain", c, "from", lo, "to", hi, "err", err)
			return
		}
		for _, l := range logs {
			msg, err := messages.MessageFromLog(l)
			if err != nil {
				k.fail(sn.Name, c, l.BlockNumber, fmt.Errorf("undecodable executing message: %w", err))
				continue
			}
			if msg == nil {
				continue
			}
			_, execTime, err := sn.Chains[c].Header(ctx, l.BlockNumber)
			if err != nil {
				k.fail(sn.Name, c, l.BlockNumber, fmt.Errorf("executing block not readable: %w", err))
				continue
			}
			st.ExecMsgs++
			if err := ValidateMessage(ctx, sn.Chains, crossSafe, execTime, msg); err != nil {
				k.fail(sn.Name, c, l.BlockNumber, err)
			}
		}
		st.CheckedTo = hi
	}
}

// Agreement compares the block hash at the lowest common cross-safe height of every
// supernode, per chain, and returns a one-line result per chain.
func (k *Checker) Agreement(ctx context.Context) map[uint64]string {
	out := map[uint64]string{}
	for _, c := range k.chains {
		height := uint64(1<<63 - 1)
		for _, sn := range k.supernodes {
			height = min(height, k.state[sn.Name][c].CrossSafe)
		}
		byHash := map[common.Hash][]string{}
		for _, sn := range k.supernodes {
			if hash, _, err := sn.Chains[c].Header(ctx, height); err == nil {
				byHash[hash] = append(byHash[hash], sn.Name)
			}
		}
		if len(byHash) <= 1 {
			out[c] = fmt.Sprintf("agree at %d", height)
			continue
		}
		var parts []string
		for h, names := range byHash {
			parts = append(parts, fmt.Sprintf("%s=%s", strings.Join(names, ","), h.Hex()[:10]))
		}
		sort.Strings(parts)
		out[c] = fmt.Sprintf("disagree at %d: %s", height, strings.Join(parts, " "))
		k.fail("all", c, height, errors.New("supernodes disagree on the cross-safe chain: "+strings.Join(parts, " ")))
	}
	return out
}

// CheckReplacements checks that GoneTxs are not canonical and DepositOnly blocks hold
// only deposits, on every supernode's ELs.
func (k *Checker) CheckReplacements(ctx context.Context) {
	for _, sn := range k.supernodes {
		for _, c := range k.chains {
			r := sn.Chains[c]
			for _, tx := range k.cfg.GoneTxs {
				if found, err := r.HasReceipt(ctx, tx); err == nil && found {
					k.fail(sn.Name, c, 0, fmt.Errorf("invalidated tx %s is still canonical", tx))
				}
			}
			for _, n := range k.cfg.DepositOnly[c] {
				types, err := r.TxTypes(ctx, n)
				if err != nil {
					continue
				}
				for _, t := range types {
					if t != depositTxType {
						k.fail(sn.Name, c, n, fmt.Errorf("replaced block holds a non-deposit tx of type %d", t))
					}
				}
			}
		}
	}
}

// Stalled returns "<supernode>/<chain>" for every chain whose cross-safe head has not
// advanced for longer than StallAfter.
func (k *Checker) Stalled() []string {
	var out []string
	now := k.now().Unix()
	for _, sn := range k.supernodes {
		for _, c := range k.chains {
			if time.Duration(now-k.state[sn.Name][c].LastAdvance)*time.Second > k.cfg.StallAfter {
				out = append(out, fmt.Sprintf("%s/%d", sn.Name, c))
			}
		}
	}
	return out
}
