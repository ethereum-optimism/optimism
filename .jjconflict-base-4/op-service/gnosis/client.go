package gnosis

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type GnosisClient struct {
	lgr           log.Logger
	ethClient     *ethclient.Client
	ChainID       *big.Int
	privateKeys   []*ecdsa.PrivateKey
	safeAddress   common.Address
	safeABI       abi.ABI
	boundContract *bind.BoundContract
	Txmgr         txmgr.TxManager
}

type SafeTransaction struct {
	To             common.Address
	Value          *big.Int
	Data           []byte
	Operation      uint8
	SafeTxGas      *big.Int
	BaseGas        *big.Int
	GasPrice       *big.Int
	GasToken       common.Address
	RefundReceiver common.Address
	Nonce          *big.Int
}

const (
	OperationCall         = 0 // Execute as regular external call
	OperationDelegateCall = 1 // Execute as delegatecall
	OperationCreate       = 2 // Create a new contract
)

var DefaultGnosisTxMgrConfig = txmgr.DefaultFlagValues{
	NumConfirmations:          uint64(2),
	SafeAbortNonceTooLowCount: uint64(3),
	FeeLimitMultiplier:        uint64(5),
	FeeLimitThresholdGwei:     100.0,
	MinTipCapGwei:             1.0,
	MinBaseFeeGwei:            1.0,
	ResubmissionTimeout:       24 * time.Second,
	NetworkTimeout:            10 * time.Second,
	RetryInterval:             1 * time.Second,
	MaxRetries:                uint64(10),
	TxSendTimeout:             2 * time.Minute,
	TxNotInMempoolTimeout:     1 * time.Minute,
	ReceiptQueryInterval:      6 * time.Second,
}

type GnosisOptions struct {
	TxMgrCLIConfig *txmgr.CLIConfig
}

type GnosisOverrides func(*GnosisOptions)

// WithCustomTxMgr allows setting specific txmgr parameters while using defaults for others
func WithCustomTxMgr(overrides func(*txmgr.CLIConfig)) GnosisOverrides {
	return func(opts *GnosisOptions) {
		if overrides != nil {
			overrides(opts.TxMgrCLIConfig)
		}
	}
}

// NewGnosisClient creates a new Gnosis Safe client
func NewGnosisClient(lgr log.Logger, rpcUrl string, privateKeys []string, safeAddress common.Address, opts ...GnosisOverrides) (*GnosisClient, error) {
	ecdsaKeys := make([]*ecdsa.PrivateKey, len(privateKeys))
	for i, key := range privateKeys {
		privateKey, err := crypto.HexToECDSA(key)
		if err != nil {
			return nil, fmt.Errorf("failed to convert private key to ECDSA: %w", err)
		}
		ecdsaKeys[i] = privateKey
	}

	// Default configuration
	cfg := &GnosisOptions{
		TxMgrCLIConfig: func() *txmgr.CLIConfig {
			defaultCfg := txmgr.NewCLIConfig(rpcUrl, DefaultGnosisTxMgrConfig)
			return &defaultCfg
		}(),
	}

	// Apply overrides
	for _, opt := range opts {
		opt(cfg)
	}

	txMgrCfg := *cfg.TxMgrCLIConfig
	txMgrCfg.PrivateKey = privateKeys[0] // Use first key for gas payments

	ethClient, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	chainID, err := ethClient.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	safeABI, err := abi.JSON(strings.NewReader(safeABIString))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Safe ABI: %w", err)
	}

	boundContract := bind.NewBoundContract(safeAddress, safeABI, ethClient, ethClient, ethClient)

	// Create the transaction manager
	txMgr, err := txmgr.NewSimpleTxManager("gnosis-safe", lgr, &metrics.NoopTxMetrics{}, txMgrCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction manager: %w", err)
	}

	gnosisClient := &GnosisClient{
		lgr:           lgr,
		ethClient:     ethClient,
		ChainID:       chainID,
		privateKeys:   ecdsaKeys,
		safeAddress:   safeAddress,
		safeABI:       safeABI,
		boundContract: boundContract,
		Txmgr:         txMgr,
	}

	err = gnosisClient.Check(context.Background())
	if err != nil {
		return nil, err
	}

	return gnosisClient, nil
}

// Check verifies that all private keys correspond to Safe owners
func (c *GnosisClient) Check(ctx context.Context) error {
	if len(c.privateKeys) == 0 {
		return fmt.Errorf("no private keys provided")
	}

	// Get Safe owners
	owners, err := c.GetOwners(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Safe owners: %w", err)
	}

	// Create a map for fast owner lookup
	ownerMap := make(map[common.Address]bool)
	for _, owner := range owners {
		ownerMap[owner] = true
	}

	// Check each private key
	for i, privateKey := range c.privateKeys {
		address := crypto.PubkeyToAddress(privateKey.PublicKey)
		if !ownerMap[address] {
			return fmt.Errorf("private keys at index %d is not Safe owner: %v", i, address)
		}
		c.lgr.Info("private key verified as Safe owner", "index", i, "address", address)
	}

	c.lgr.Info("all private keys verified as Safe owners", "numKeys", len(c.privateKeys), "safeAddress", c.safeAddress)
	return nil
}

// CreateTransaction creates a tx from the Safe
func (c *GnosisClient) CreateTransaction(ctx context.Context, to common.Address, value *big.Int, calldata []byte, operation uint8) (*SafeTransaction, error) {
	nonce, err := c.GetNonce(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	safeTx := &SafeTransaction{
		To:             to,
		Value:          value,
		Data:           calldata,
		Operation:      operation,
		SafeTxGas:      big.NewInt(0),
		BaseGas:        big.NewInt(0),
		GasPrice:       big.NewInt(0),
		GasToken:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
		RefundReceiver: common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Nonce:          nonce,
	}

	c.lgr.Info("created transaction")
	return safeTx, nil
}

// SignTransaction signs a Safe transaction with all private keys
func (c *GnosisClient) SignTransaction(safeTx *SafeTransaction) ([]byte, error) {
	txHash, err := c.getTransactionHash(safeTx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction hash: %w", err)
	}

	var signatures []byte
	for i, privateKey := range c.privateKeys {
		signature, err := crypto.Sign(txHash, privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign with key %d: %w", i, err)
		}

		// Safe uses a different recovery ID format
		// For ECDSA signatures, v should be 27 or 28
		if signature[64] < 27 {
			signature[64] += 27
		}

		signatures = append(signatures, signature...)
	}

	c.lgr.Info("signed tx with all keys")
	return signatures, nil
}

// ExecuteTransaction executes a Safe transaction using txmgr for reliability
func (c *GnosisClient) ExecuteTransaction(ctx context.Context, safeTx *SafeTransaction, signatures []byte) (*types.Receipt, error) {
	// Check that the number of signatures meets the safe threshold
	threshold, err := c.getThreshold(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get threshold: %w", err)
	}
	numSignatures := len(signatures) / 65
	if numSignatures < int(threshold.Uint64()) {
		return nil, fmt.Errorf("not enough signatures to execute transaction: have %d, need %d",
			numSignatures, threshold.Uint64())
	}
	c.lgr.Info("signatures meets threshold", "numSignatures", numSignatures, "threshold", threshold)

	// Create calldata for execTransaction
	calldata, err := c.safeABI.Pack("execTransaction",
		safeTx.To,
		safeTx.Value,
		safeTx.Data,
		safeTx.Operation,
		safeTx.SafeTxGas,
		safeTx.BaseGas,
		safeTx.GasPrice,
		safeTx.GasToken,
		safeTx.RefundReceiver,
		signatures,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack transaction data: %w", err)
	}

	// Create TxCandidate for txmgr
	candidate := txmgr.TxCandidate{
		TxData:   calldata,
		To:       &c.safeAddress,
		Value:    big.NewInt(0), // execTransaction doesn't send ETH
		GasLimit: 0,             // Let txmgr estimate gas
	}

	c.lgr.Info("executing Safe transaction via txmgr",
		"safeAddress", c.safeAddress.Hex(),
		"to", safeTx.To.Hex(),
		"value", safeTx.Value.String(),
		"operation", safeTx.Operation,
		"numSignatures", numSignatures)

	// Use txmgr to send the transaction with automatic retries and fee bumping
	receipt, err := c.Txmgr.Send(ctx, candidate)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Safe transaction: %w", err)
	}

	c.lgr.Info("Safe transaction executed successfully",
		"txHash", receipt.TxHash.Hex(),
		"gasUsed", receipt.GasUsed,
		"effectiveGasPrice", receipt.EffectiveGasPrice)

	return receipt, nil
}

func (c *GnosisClient) SendTransaction(ctx context.Context, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Receipt, error) {
	tx, err := c.CreateTransaction(ctx, to, value, data, operation)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	signature, err := c.SignTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	receipt, err := c.ExecuteTransaction(ctx, tx, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transaction: %w", err)
	}

	return receipt, nil
}

func (c *GnosisClient) IsOwner(ctx context.Context, address common.Address) (bool, error) {
	var result []interface{}
	err := c.boundContract.Call(&bind.CallOpts{Context: ctx}, &result, "isOwner", address)
	if err != nil {
		return false, fmt.Errorf("failed to check owner: %w", err)
	}
	return result[0].(bool), nil
}

func (c *GnosisClient) GetOwners(ctx context.Context) ([]common.Address, error) {
	var result []interface{}
	err := c.boundContract.Call(&bind.CallOpts{Context: ctx}, &result, "getOwners")
	if err != nil {
		return nil, fmt.Errorf("failed to get owners: %w", err)
	}
	return result[0].([]common.Address), nil
}

func (c *GnosisClient) GetNonce(ctx context.Context) (*big.Int, error) {
	var result []interface{}
	err := c.boundContract.Call(&bind.CallOpts{Context: ctx}, &result, "nonce")
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	return result[0].(*big.Int), nil
}

// Close closes the underlying connections
func (c *GnosisClient) Close() {
	if c.Txmgr != nil {
		c.Txmgr.Close()
	}
	if c.ethClient != nil {
		c.ethClient.Close()
	}
}

func (c *GnosisClient) getThreshold(ctx context.Context) (*big.Int, error) {
	var result []interface{}
	err := c.boundContract.Call(&bind.CallOpts{Context: ctx}, &result, "getThreshold")
	if err != nil {
		return nil, fmt.Errorf("failed to get threshold: %w", err)
	}

	return result[0].(*big.Int), nil
}

func (c *GnosisClient) getTransactionHash(safeTx *SafeTransaction) ([]byte, error) {
	// Define the EIP-712 typed data
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"SafeTx": []apitypes.Type{
				{Name: "to", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "data", Type: "bytes"},
				{Name: "operation", Type: "uint8"},
				{Name: "safeTxGas", Type: "uint256"},
				{Name: "baseGas", Type: "uint256"},
				{Name: "gasPrice", Type: "uint256"},
				{Name: "gasToken", Type: "address"},
				{Name: "refundReceiver", Type: "address"},
				{Name: "nonce", Type: "uint256"},
			},
		},
		PrimaryType: "SafeTx",
		Domain: apitypes.TypedDataDomain{
			ChainId:           (*math.HexOrDecimal256)(c.ChainID),
			VerifyingContract: c.safeAddress.Hex(),
		},
		Message: map[string]interface{}{
			"to":             safeTx.To.Hex(),
			"value":          safeTx.Value.String(),
			"data":           "0x" + hex.EncodeToString(safeTx.Data),
			"operation":      fmt.Sprintf("%d", safeTx.Operation),
			"safeTxGas":      safeTx.SafeTxGas.String(),
			"baseGas":        safeTx.BaseGas.String(),
			"gasPrice":       safeTx.GasPrice.String(),
			"gasToken":       safeTx.GasToken.Hex(),
			"refundReceiver": safeTx.RefundReceiver.Hex(),
			"nonce":          safeTx.Nonce.String(),
		},
	}

	// Calculate the hash using go-ethereum's EIP-712 implementation
	hash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate EIP-712 hash: %w", err)
	}

	return hash, nil
}
