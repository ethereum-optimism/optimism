package crossdomain

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// A PendingWithdrawal represents a withdrawal that has
// not been finalized on L1
type PendingWithdrawal struct {
	LegacyWithdrawal `json:"withdrawal"`
	TransactionHash  common.Hash `json:"transactionHash"`
}

// Backends represents a set of backends for L1 and L2.
// These are used as the backends for the Messengers
type Backends struct {
	L1 bind.ContractBackend
	L2 bind.ContractBackend
}

func NewBackends(l1, l2 bind.ContractBackend) *Backends {
	return &Backends{
		L1: l1,
		L2: l2,
	}
}

// Messengers represents a pair of L1 and L2 cross domain messengers
// that are connected to the correct contract addresses
type Messengers struct {
	L1 *bindings.L1CrossDomainMessenger
	L2 *bindings.L2CrossDomainMessenger
}

// NewMessengers constructs Messengers. Passing in the address of the
// L1CrossDomainMessenger is required to connect to the
func NewMessengers(backends *Backends, l1CrossDomainMessenger common.Address) (*Messengers, error) {
	l1Messenger, err := bindings.NewL1CrossDomainMessenger(l1CrossDomainMessenger, backends.L1)
	if err != nil {
		return nil, err
	}
	l2Messenger, err := bindings.NewL2CrossDomainMessenger(predeploys.L2CrossDomainMessengerAddr, backends.L2)
	if err != nil {
		return nil, err
	}

	return &Messengers{
		L1: l1Messenger,
		L2: l2Messenger,
	}, nil
}

// GetPendingWithdrawals will fetch pending withdrawals by getting
// L2CrossDomainMessenger `SentMessage` events and then checking to see if the
// cross domain message hash has been finalized on L1. It will return a slice of
// PendingWithdrawals that have not been finalized on L1.
func GetPendingWithdrawals(messengers *Messengers, version *big.Int, start, end uint64) ([]PendingWithdrawal, error) {
	withdrawals := make([]PendingWithdrawal, 0)

	// This will not take into account "pending" state, this ensures that
	// transactions in the mempool are upgraded as well.
	opts := bind.FilterOpts{
		Start: start,
	}
	// Only set the end block range if end is non zero. When end is zero, the
	// filter will extend to the latest block.
	if end != 0 {
		opts.End = &end
	}

	messages, err := messengers.L2.FilterSentMessage(&opts, nil)
	if err != nil {
		return nil, err
	}

	defer messages.Close()
	for messages.Next() {
		event := messages.Event

		msg := NewCrossDomainMessage(
			event.MessageNonce,
			event.Sender,
			event.Target,
			common.Big0,
			event.GasLimit,
			event.Message,
		)

		// Optional version check
		if version != nil {
			if version.Uint64() != msg.Version() {
				return nil, fmt.Errorf("expected version %d, got version %d", version, msg.Version())
			}
		}

		hash, err := msg.Hash()
		if err != nil {
			return nil, err
		}

		relayed, err := messengers.L1.SuccessfulMessages(&bind.CallOpts{}, hash)
		if err != nil {
			return nil, err
		}

		if !relayed {
			log.Info("%s not yet relayed", event.Raw.TxHash)

			withdrawal := PendingWithdrawal{
				LegacyWithdrawal: LegacyWithdrawal{
					XDomainTarget: event.Target,
					XDomainSender: event.Sender,
					XDomainData:   event.Message,
					XDomainNonce:  event.MessageNonce,
				},
				TransactionHash: event.Raw.TxHash,
			}

			withdrawals = append(withdrawals, withdrawal)
		} else {
			log.Info("%s already relayed", event.Raw.TxHash)
		}
	}
	return withdrawals, nil
}
