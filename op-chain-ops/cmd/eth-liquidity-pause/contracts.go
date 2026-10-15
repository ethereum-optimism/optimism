package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// syncSignature is the permissionless L1CrossDomainMessenger entrypoint that reads the chain's
// L1 pause (SuperchainConfig, scoped to the chain's ETHLockbox) and sends
// ETHLiquidity.setPaused(paused, messageNonce) to L2, where messageNonce is the versioned nonce
// of that cross-domain message.
const syncSignature = "syncETHLiquidityPause()"

var (
	// L2 predeploys.
	pausedFn             = w3.MustNewFunc("paused()", "bool")
	pausedMessageNonceFn = w3.MustNewFunc("pausedMessageNonce()", "uint256")
	setPausedFn          = w3.MustNewFunc("setPaused(bool,uint256)", "")
	sendETHFn            = w3.MustNewFunc("sendETH(address,uint256)", "bytes32")
	failedMessagesFn     = w3.MustNewFunc("failedMessages(bytes32)", "bool")

	// L1CrossDomainMessenger.
	syncFn         = w3.MustNewFunc(syncSignature, "")
	systemConfigFn = w3.MustNewFunc("systemConfig()", "address")
	sentMessageEvt = w3.MustNewEvent("SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)")

	// SystemConfig.
	l2ChainIDFn      = w3.MustNewFunc("l2ChainId()", "uint256")
	optimismPortalFn = w3.MustNewFunc("optimismPortal()", "address")

	// OptimismPortal2.
	ethLockboxFn = w3.MustNewFunc("ethLockbox()", "address")

	// ETHLockbox.
	superchainConfigFn = w3.MustNewFunc("superchainConfig()", "address")

	// SuperchainConfig.
	guardianFn         = w3.MustNewFunc("guardian()", "address")
	pausedForFn        = w3.MustNewFunc("paused(address)", "bool")
	pausableFn         = w3.MustNewFunc("pausable(address)", "bool")
	scPauseFn          = w3.MustNewFunc("pause(address)", "")
	scUnpauseFn        = w3.MustNewFunc("unpause(address)", "")
	scExtendFn         = w3.MustNewFunc("extend(address)", "")
	globalPauseAddress = common.Address{}
)

// knownErrors maps revert selectors to the custom errors the sendETH path and the L1 pause
// calls can raise, so reverts print by name.
var knownErrors = func() map[[4]byte]string {
	m := make(map[[4]byte]string)
	for _, sig := range []string{
		"ETHLiquidity_Paused()",
		"ETHLiquidity_NotL1CrossDomainMessenger()",
		"L1CrossDomainMessenger_InteropNotEnabled()",
		"Unauthorized()",
		"ZeroAddress()",
		"InvalidAmount()",
		"MessageDestinationSameChain()",
		"L2ToL2CrossDomainMessenger_MessageTargetUnsafe()",
		"SuperchainConfig_OnlyGuardian()",
		"SuperchainConfig_AlreadyPaused(address)",
		"SuperchainConfig_NotAlreadyPaused(address)",
		errorSignature,
		panicSignature,
	} {
		m[selector(sig)] = sig
	}
	return m
}()

// pausedErrorName is the error ETHLiquidity raises from burn() and mint() while paused.
const pausedErrorName = "ETHLiquidity_Paused()"

// The Solidity built-in errors: require/revert reasons and compiler-inserted panics.
const (
	errorSignature = "Error(string)"
	panicSignature = "Panic(uint256)"
)

var (
	errorSelector = selector(errorSignature)
	panicSelector = selector(panicSignature)
)

func selector(sig string) [4]byte {
	var s [4]byte
	copy(s[:], crypto.Keccak256([]byte(sig))[:4])
	return s
}

// revertInfo describes the revert of an eth_call.
type revertInfo struct {
	// Selector is the 0x-prefixed 4-byte selector, or empty if the revert carried no data.
	Selector string `json:"selector,omitempty"`
	// Name is the decoded error signature, if the selector is known.
	Name string `json:"name,omitempty"`
	// Message is the raw node error message.
	Message string `json:"message"`
}

func (r *revertInfo) String() string {
	switch {
	case r.Name != "":
		return fmt.Sprintf("%s (%s)", r.Name, r.Selector)
	case r.Selector != "":
		return r.Selector
	default:
		return r.Message
	}
}

// dataError is implemented by geth RPC errors that carry revert data.
type dataError interface {
	error
	ErrorData() interface{}
}

// decodeRevert extracts the revert selector from an eth_call error. It returns nil if err does
// not look like an execution revert (e.g. a transport failure).
func decodeRevert(err error) *revertInfo {
	if err == nil {
		return nil
	}
	var de dataError
	if !errors.As(err, &de) {
		if strings.Contains(err.Error(), "execution reverted") {
			return &revertInfo{Message: err.Error()}
		}
		return nil
	}
	info := &revertInfo{Message: de.Error()}
	s, ok := de.ErrorData().(string)
	if !ok {
		return info
	}
	data, decErr := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if decErr != nil || len(data) < 4 {
		return info
	}
	var sel [4]byte
	copy(sel[:], data[:4])
	info.Selector = "0x" + hex.EncodeToString(sel[:])
	info.Name = knownErrors[sel]
	switch sel {
	case errorSelector:
		if args, err := (abi.Arguments{{Type: stringType}}).Unpack(data[4:]); err == nil {
			info.Name = fmt.Sprintf("Error(%q)", args[0])
		}
	case panicSelector:
		if args, err := (abi.Arguments{{Type: uint256Type}}).Unpack(data[4:]); err == nil {
			info.Name = fmt.Sprintf("Panic(%#x)", args[0])
		}
	}
	return info
}

var (
	stringType  = mustABIType("string")
	uint256Type = mustABIType("uint256")
)

func mustABIType(t string) abi.Type {
	typ, err := abi.NewType(t, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}

// callView performs an eth_call of fn on to and decodes the result into out. A nil block reads
// the latest state.
func callView(ctx context.Context, cl ethereum.ContractCaller, to common.Address, block *big.Int, fn *w3.Func, out any, args ...any) error {
	data, err := fn.EncodeArgs(args...)
	if err != nil {
		return err
	}
	res, err := cl.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, block)
	if err != nil {
		return err
	}
	return fn.DecodeReturns(res, out)
}

// l1Contracts are the L1 contracts that hold one chain's pause state.
type l1Contracts struct {
	Messenger        common.Address `json:"l1CrossDomainMessenger"`
	SystemConfig     common.Address `json:"systemConfig"`
	Portal           common.Address `json:"optimismPortal"`
	Lockbox          common.Address `json:"ethLockbox"`
	SuperchainConfig common.Address `json:"superchainConfig"`
	L2ChainID        uint64         `json:"l2ChainId"`
}

// resolveL1 walks L1CrossDomainMessenger -> SystemConfig -> OptimismPortal -> ETHLockbox ->
// SuperchainConfig. The ETHLockbox address is the SuperchainConfig pause identifier.
func resolveL1(ctx context.Context, l1 ethereum.ContractCaller, messenger common.Address) (*l1Contracts, error) {
	c := &l1Contracts{Messenger: messenger}
	if err := callView(ctx, l1, messenger, nil, systemConfigFn, &c.SystemConfig); err != nil {
		return nil, fmt.Errorf("L1CrossDomainMessenger %s systemConfig(): %w", messenger, err)
	}
	var chainID big.Int
	if err := callView(ctx, l1, c.SystemConfig, nil, l2ChainIDFn, &chainID); err != nil {
		return nil, fmt.Errorf("SystemConfig %s l2ChainId(): %w", c.SystemConfig, err)
	}
	if !chainID.IsUint64() {
		return nil, fmt.Errorf("SystemConfig %s l2ChainId() %s does not fit in uint64", c.SystemConfig, &chainID)
	}
	c.L2ChainID = bigs.Uint64Strict(&chainID)
	if err := callView(ctx, l1, c.SystemConfig, nil, optimismPortalFn, &c.Portal); err != nil {
		return nil, fmt.Errorf("SystemConfig %s optimismPortal(): %w", c.SystemConfig, err)
	}
	if err := callView(ctx, l1, c.Portal, nil, ethLockboxFn, &c.Lockbox); err != nil {
		return nil, fmt.Errorf("OptimismPortal %s ethLockbox(): %w", c.Portal, err)
	}
	if c.Lockbox == (common.Address{}) {
		return nil, fmt.Errorf("OptimismPortal %s has no ETHLockbox", c.Portal)
	}
	if err := callView(ctx, l1, c.Lockbox, nil, superchainConfigFn, &c.SuperchainConfig); err != nil {
		return nil, fmt.Errorf("ETHLockbox %s superchainConfig(): %w", c.Lockbox, err)
	}
	return c, nil
}

// l1Paused reads L1CrossDomainMessenger.paused(), which is the ETHLockbox pause (the global or
// lockbox-scoped SuperchainConfig pause) that the sync mirrors to L2.
func l1Paused(ctx context.Context, l1 ethereum.ContractCaller, messenger common.Address, block *big.Int) (bool, error) {
	var paused bool
	if err := callView(ctx, l1, messenger, block, pausedFn, &paused); err != nil {
		return false, fmt.Errorf("L1CrossDomainMessenger %s paused(): %w", messenger, err)
	}
	return paused, nil
}
