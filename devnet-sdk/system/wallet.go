package system

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	supervisorTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	coreTypes "github.com/ethereum/go-ethereum/core/types"
)

var (
	// This will make sure that we implement the Chain interface
	_ Wallet = (*wallet)(nil)
)

// internalChain provides access to internal chain functionality
type internalChain interface {
	Chain
	Client() (*sources.EthClient, error)
	GethClient() (*ethclient.Client, error)
}

type wallet struct {
	privateKey types.Key
	address    types.Address
	chain      internalChain
}

func newWalletMapFromDescriptorWalletMap(descriptorWalletMap descriptors.WalletMap, chain internalChain) (WalletMap, error) {
	result := WalletMap{}
	for k, v := range descriptorWalletMap {
		wallet, err := newWallet(v.PrivateKey, v.Address, chain)
		if err != nil {
			return nil, err
		}
		result[k] = wallet
	}
	return result, nil
}

func newWallet(pk string, addr types.Address, chain internalChain) (*wallet, error) {
	privateKey, err := privateKeyFromString(pk)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private from string: %w", err)
	}

	return &wallet{
		privateKey: privateKey,
		address:    addr,
		chain:      chain,
	}, nil
}

func privateKeyFromString(pk string) (types.Key, error) {
	var privateKey types.Key
	if pk != "" {
		pk = strings.TrimPrefix(pk, "0x")
		if len(pk)%2 == 1 {
			pk = "0" + pk
		}
		pkBytes, err := hex.DecodeString(pk)
		if err != nil {
			return nil, fmt.Errorf("failed to decode private key: %w", err)
		}
		key, err := crypto.ToECDSA(pkBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to convert private key to ECDSA: %w", err)
		}
		privateKey = key
	}

	return privateKey, nil
}

func (w *wallet) PrivateKey() types.Key {
	return w.privateKey
}

func (w *wallet) Address() types.Address {
	return w.address
}

func (w *wallet) SendETH(to types.Address, amount types.Balance) types.WriteInvocation[any] {
	return &sendImpl{
		chain:     w.chain,
		processor: w,
		from:      w.address,
		to:        to,
		amount:    amount,
	}
}

func (w *wallet) Balance() types.Balance {
	client, err := w.chain.Client()
	if err != nil {
		return types.Balance{}
	}

	balance, err := client.BalanceAt(context.Background(), w.address, nil)
	if err != nil {
		return types.Balance{}
	}

	return types.NewBalance(balance)
}

func (w *wallet) InitiateMessage(chainID types.ChainID, target common.Address, message []byte) types.WriteInvocation[any] {
	return &initiateMessageImpl{
		chain:     w.chain,
		processor: w,
		from:      w.address,
		target:    target,
		chainID:   chainID,
		message:   message,
	}
}

func (w *wallet) ExecuteMessage(identifier bindings.Identifier, sentMessage []byte) types.WriteInvocation[any] {
	return &executeMessageImpl{
		chain:       w.chain,
		processor:   w,
		from:        w.address,
		identifier:  identifier,
		sentMessage: sentMessage,
	}
}

type initiateMessageImpl struct {
	chain     internalChain
	processor TransactionProcessor
	from      types.Address

	target  types.Address
	chainID types.ChainID
	message []byte
}

func (i *initiateMessageImpl) Call(ctx context.Context) (any, error) {
	builder := NewTxBuilder(ctx, i.chain)
	messenger, err := i.chain.ContractsRegistry().L2ToL2CrossDomainMessenger(constants.L2ToL2CrossDomainMessenger)
	if err != nil {
		return nil, fmt.Errorf("failed to init transaction: %w", err)
	}
	data, err := messenger.ABI().Pack("sendMessage", i.chainID, i.target, i.message)
	if err != nil {
		return nil, fmt.Errorf("failed to build calldata: %w", err)
	}
	tx, err := builder.BuildTx(
		WithFrom(i.from),
		WithTo(constants.L2ToL2CrossDomainMessenger),
		WithValue(big.NewInt(0)),
		WithData(data),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	tx, err = i.processor.Sign(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return tx, nil
}

func (i *initiateMessageImpl) Send(ctx context.Context) types.InvocationResult {
	result, err := i.Call(ctx)
	if err != nil {
		return &sendResult{chain: i.chain, tx: nil, err: err}
	}
	tx, ok := result.(Transaction)
	if !ok {
		return &sendResult{chain: i.chain, tx: nil, err: fmt.Errorf("unexpected return type")}
	}
	err = i.processor.Send(ctx, tx)
	return &sendResult{
		chain: i.chain,
		tx:    tx,
		err:   err,
	}
}

type executeMessageImpl struct {
	chain     internalChain
	processor TransactionProcessor
	from      types.Address

	identifier  bindings.Identifier
	sentMessage []byte
}

func (i *executeMessageImpl) Call(ctx context.Context) (any, error) {
	builder := NewTxBuilder(ctx, i.chain)
	messenger, err := i.chain.ContractsRegistry().L2ToL2CrossDomainMessenger(constants.L2ToL2CrossDomainMessenger)
	if err != nil {
		return nil, fmt.Errorf("failed to init transaction: %w", err)
	}
	data, err := messenger.ABI().Pack("relayMessage", i.identifier, i.sentMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to build calldata: %w", err)
	}
	// Wrapper to use Access implementation
	msg := supervisorTypes.Message{
		Identifier: supervisorTypes.Identifier{
			Origin:      i.identifier.Origin,
			BlockNumber: i.identifier.BlockNumber.Uint64(),
			LogIndex:    uint32(i.identifier.LogIndex.Uint64()),
			Timestamp:   i.identifier.Timestamp.Uint64(),
			ChainID:     eth.ChainIDFromBig(i.identifier.ChainId),
		},
		PayloadHash: crypto.Keccak256Hash(i.sentMessage),
	}
	access := msg.Access()
	accessList := coreTypes.AccessList{{
		Address:     constants.CrossL2Inbox,
		StorageKeys: supervisorTypes.EncodeAccessList([]supervisorTypes.Access{access}),
	}}
	tx, err := builder.BuildTx(
		WithFrom(i.from),
		WithTo(constants.L2ToL2CrossDomainMessenger),
		WithValue(big.NewInt(0)),
		WithData(data),
		WithAccessList(accessList),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}
	tx, err = i.processor.Sign(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	return tx, nil
}

func (i *executeMessageImpl) Send(ctx context.Context) types.InvocationResult {
	result, err := i.Call(ctx)
	if err != nil {
		return &sendResult{chain: i.chain, tx: nil, err: err}
	}
	tx, ok := result.(Transaction)
	if !ok {
		return &sendResult{chain: i.chain, tx: nil, err: fmt.Errorf("unexpected return type")}
	}
	err = i.processor.Send(ctx, tx)
	return &sendResult{
		chain: i.chain,
		tx:    tx,
		err:   err,
	}
}

func (w *wallet) Nonce() uint64 {
	client, err := w.chain.Client()
	if err != nil {
		return 0
	}

	nonce, err := client.PendingNonceAt(context.Background(), w.address)
	if err != nil {
		return 0
	}

	return nonce
}

func (w *wallet) Transactor() *bind.TransactOpts {
	transactor, err := bind.NewKeyedTransactorWithChainID(w.PrivateKey(), w.chain.ID())
	if err != nil {
		panic(fmt.Sprintf("could not create transactor for address %s and chainID %v", w.Address(), w.chain.ID()))
	}

	return transactor
}

func (w *wallet) Sign(tx Transaction) (Transaction, error) {
	pk := w.privateKey

	var signer coreTypes.Signer
	switch tx.Type() {
	case coreTypes.SetCodeTxType:
		signer = coreTypes.NewIsthmusSigner(w.chain.ID())
	case coreTypes.DynamicFeeTxType:
		signer = coreTypes.NewLondonSigner(w.chain.ID())
	case coreTypes.AccessListTxType:
		signer = coreTypes.NewEIP2930Signer(w.chain.ID())
	default:
		signer = coreTypes.NewEIP155Signer(w.chain.ID())
	}

	if rt, ok := tx.(RawTransaction); ok {
		signedTx, err := coreTypes.SignTx(rt.Raw(), signer, pk)
		if err != nil {
			return nil, fmt.Errorf("failed to sign transaction: %w", err)
		}

		return &EthTx{
			tx:     signedTx,
			from:   tx.From(),
			txType: tx.Type(),
		}, nil
	}

	return nil, fmt.Errorf("transaction does not support signing")
}

func (w *wallet) Send(ctx context.Context, tx Transaction) error {
	if st, ok := tx.(RawTransaction); ok {
		client, err := w.chain.Client()
		if err != nil {
			return fmt.Errorf("failed to get client: %w", err)
		}
		if err := client.SendTransaction(ctx, st.Raw()); err != nil {
			return fmt.Errorf("failed to send transaction: %w", err)
		}
		return nil
	}

	return fmt.Errorf("transaction is not signed")
}

type sendImpl struct {
	chain     internalChain
	processor TransactionProcessor
	from      types.Address
	to        types.Address
	amount    types.Balance
}

func (i *sendImpl) Call(ctx context.Context) (any, error) {
	builder := NewTxBuilder(ctx, i.chain)
	tx, err := builder.BuildTx(
		WithFrom(i.from),
		WithTo(i.to),
		WithValue(i.amount.Int),
		WithData(nil),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	tx, err = i.processor.Sign(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return tx, nil
}

func (i *sendImpl) Send(ctx context.Context) types.InvocationResult {
	builder := NewTxBuilder(ctx, i.chain)
	tx, err := builder.BuildTx(
		WithFrom(i.from),
		WithTo(i.to),
		WithValue(i.amount.Int),
		WithData(nil),
	)

	// Sign the transaction if it's built okay
	if err == nil {
		tx, err = i.processor.Sign(tx)
	}

	// Send the transaction if it's signed okay
	if err == nil {
		err = i.processor.Send(ctx, tx)
	}

	return &sendResult{
		chain: i.chain,
		tx:    tx,
		err:   err,
	}
}

type sendResult struct {
	chain   internalChain
	tx      Transaction
	receipt Receipt
	err     error
}

func (r *sendResult) Error() error {
	return r.err
}

func (r *sendResult) Wait() error {
	client, err := r.chain.GethClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if r.err != nil {
		return r.err
	}
	if r.tx == nil {
		return fmt.Errorf("no transaction to wait for")
	}

	if tx, ok := r.tx.(RawTransaction); ok {
		receipt, err := wait.ForReceiptOK(context.Background(), client, tx.Raw().Hash())
		if err != nil {
			return fmt.Errorf("failed waiting for transaction confirmation: %w", err)
		}
		r.receipt = &EthReceipt{blockNumber: receipt.BlockNumber, logs: receipt.Logs, txHash: receipt.TxHash}
		if receipt.Status == 0 {
			return fmt.Errorf("transaction failed")
		}
	}

	return nil
}

func (r *sendResult) Info() any {
	return r.receipt
}
