package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// ExpiryNoticeResult is the tuple returned by the MessageExpiryHub's `notices` mapping getter.
// A zero AttestedAt means no notice is recorded.
type ExpiryNoticeResult struct {
	AnchorStateRegistry common.Address
	AttestedAt          uint64
}

// MessageExpiryHub binds the ownerless MessageExpiryHub L1 singleton. No deployment pipeline
// installs it yet, so tests deploy it themselves from MessageExpiryHubBin.
type MessageExpiryHub struct {
	// Read-only functions. Notices are keyed by the attestor's shared ETHLockbox (its cluster
	// identity), the attestor chain ID, the message's source chain ID, and the message hash.
	Notices func(
		ethLockbox common.Address,
		attestorChainID eth.ChainID,
		sourceChainID eth.ChainID,
		msgHash [32]byte,
	) TypedCall[ExpiryNoticeResult] `sol:"notices"`
	// RegisteredChains returns the registered chain's SystemConfig, or the zero address.
	RegisteredChains func(ethLockbox common.Address, chainID eth.ChainID) TypedCall[common.Address] `sol:"registeredChains"`
	Version          func() TypedCall[string]                                                       `sol:"version"`

	// Write functions
	RegisterChain       func(systemConfig common.Address) TypedCall[any] `sol:"registerChain"`
	ForwardExpiryNotice func(
		ethLockbox common.Address,
		attestorChainID eth.ChainID,
		sourceChainID eth.ChainID,
		msgHash [32]byte,
		minGasLimit uint32,
	) TypedCall[any] `sol:"forwardExpiryNotice"`
}
