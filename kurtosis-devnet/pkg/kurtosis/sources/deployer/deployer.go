package deployer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"text/template"

	ktfs "github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	defaultDeployerArtifactName = "op-deployer-configs"
	defaultWalletsName          = "wallets.json"
	defaultStateName            = "state.json"
	defaultGenesisArtifactName  = "el_cl_genesis_data"
	defaultMnemonicName         = "mnemonics.yaml"
	defaultGenesisNameTemplate  = "genesis-{{.ChainID}}.json"
	defaultL1GenesisName        = "genesis.json"
)

// DeploymentAddresses maps contract names to their addresses
type DeploymentAddresses map[string]types.Address

// DeploymentStateAddresses maps chain IDs to their contract addresses
type DeploymentStateAddresses map[string]DeploymentAddresses

type DeploymentState struct {
	L1Addresses DeploymentAddresses `json:"l1_addresses"`
	L2Addresses DeploymentAddresses `json:"l2_addresses"`
	L1Wallets   WalletList          `json:"l1_wallets"`
	L2Wallets   WalletList          `json:"l2_wallets"`
	Config      *params.ChainConfig `json:"chain_config"`
}

type DeployerState struct {
	Deployments map[string]DeploymentState `json:"l2s"`
	Addresses   DeploymentAddresses        `json:"superchain"`
}

// StateFile represents the structure of the state.json file
type StateFile struct {
	OpChainDeployments        []map[string]interface{} `json:"opChainDeployments"`
	SuperChainDeployment      map[string]interface{}   `json:"superchainDeployment"`
	ImplementationsDeployment map[string]interface{}   `json:"implementationsDeployment"`
}

// Wallet represents a wallet with optional private key and name
type Wallet struct {
	Address    types.Address `json:"address"`
	PrivateKey string        `json:"private_key"`
	Name       string        `json:"name"`
}

// WalletList holds a list of wallets
type WalletList []*Wallet
type WalletMap map[string]*Wallet

type DeployerData struct {
	L1ValidatorWallets WalletList     `json:"wallets"`
	State              *DeployerState `json:"state"`
	L1ChainID          string         `json:"l1_chain_id"`
}

type Deployer struct {
	enclave                 string
	deployerArtifactName    string
	walletsName             string
	stateName               string
	genesisArtifactName     string
	l1ValidatorMnemonicName string
	l2GenesisNameTemplate   string
	l1GenesisName           string
}

type DeployerOption func(*Deployer)

func WithArtifactName(name string) DeployerOption {
	return func(d *Deployer) {
		d.deployerArtifactName = name
	}
}

func WithWalletsName(name string) DeployerOption {
	return func(d *Deployer) {
		d.walletsName = name
	}
}

func WithStateName(name string) DeployerOption {
	return func(d *Deployer) {
		d.stateName = name
	}
}

func WithGenesisArtifactName(name string) DeployerOption {
	return func(d *Deployer) {
		d.genesisArtifactName = name
	}
}

func WithMnemonicsName(name string) DeployerOption {
	return func(d *Deployer) {
		d.l1ValidatorMnemonicName = name
	}
}

func WithGenesisNameTemplate(name string) DeployerOption {
	return func(d *Deployer) {
		d.l2GenesisNameTemplate = name
	}
}

func NewDeployer(enclave string, opts ...DeployerOption) *Deployer {
	d := &Deployer{
		enclave:                 enclave,
		deployerArtifactName:    defaultDeployerArtifactName,
		walletsName:             defaultWalletsName,
		stateName:               defaultStateName,
		genesisArtifactName:     defaultGenesisArtifactName,
		l1ValidatorMnemonicName: defaultMnemonicName,
		l2GenesisNameTemplate:   defaultGenesisNameTemplate,
		l1GenesisName:           defaultL1GenesisName,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// parseWalletsFile parses a JSON file containing wallet information
func parseWalletsFile(r io.Reader) (map[string]WalletList, error) {
	result := make(map[string]WalletList)

	// Read all data from reader
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file: %w", err)
	}

	// Unmarshal into a map first
	var rawData map[string]map[string]string
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to decode wallet file: %w", err)
	}

	for id, chain := range rawData {
		// Create a map to store wallets by name
		walletMap := make(WalletMap)
		hasAddress := make(map[string]bool)

		// First pass: collect addresses
		for key, value := range chain {
			if strings.HasSuffix(key, "Address") {
				name := strings.TrimSuffix(key, "Address")
				wallet, ok := walletMap[name]
				if !ok || wallet == nil {
					wallet = &Wallet{
						Name:    name,
						Address: common.HexToAddress(value),
					}
				} else {
					log.Warn("duplicate wallet name key in wallets file", "name", name)
				}
				walletMap[name] = wallet
				hasAddress[name] = true
			}
		}

		// Second pass: collect private keys only for wallets with addresses
		for key, value := range chain {
			if strings.HasSuffix(key, "PrivateKey") {
				name := strings.TrimSuffix(key, "PrivateKey")
				if hasAddress[name] {
					wallet := walletMap[name]
					wallet.PrivateKey = value
					walletMap[name] = wallet
				}
			}
		}

		// Convert map to list, only including wallets with addresses
		wl := make(WalletList, 0, len(walletMap))
		for name, wallet := range walletMap {
			if hasAddress[name] {
				wl = append(wl, wallet)
			}
		}

		result[id] = wl
	}

	return result, nil
}

// hexToDecimal converts a hex string (with or without 0x prefix) to a decimal string
func hexToDecimal(hex string) (string, error) {
	// Remove 0x prefix if present
	hex = strings.TrimPrefix(hex, "0x")

	// Parse hex string to big.Int
	n := new(big.Int)
	if _, ok := n.SetString(hex, 16); !ok {
		return "", fmt.Errorf("invalid hex string: %s", hex)
	}

	// Convert to decimal string
	return n.String(), nil
}

// parseStateFile parses the state.json file and extracts addresses
func parseStateFile(r io.Reader) (*DeployerState, error) {
	var state StateFile
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode state file: %w", err)
	}

	result := &DeployerState{
		Deployments: make(map[string]DeploymentState),
		Addresses:   make(DeploymentAddresses),
	}

	mapDeployment := func(deployment map[string]interface{}) DeploymentAddresses {
		addrSuffix := "Address"
		addresses := make(DeploymentAddresses)
		for key, value := range deployment {
			if strings.HasSuffix(key, addrSuffix) {
				addresses[strings.TrimSuffix(key, addrSuffix)] = common.HexToAddress(value.(string))
			}
		}
		return addresses
	}

	for _, deployment := range state.OpChainDeployments {
		// Get the chain ID
		idValue, ok := deployment["id"]
		if !ok {
			continue
		}
		hexID, ok := idValue.(string)
		if !ok {
			continue
		}

		// Convert hex ID to decimal
		id, err := hexToDecimal(hexID)
		if err != nil {
			continue
		}

		l1Addresses := mapDeployment(deployment)

		// op-deployer currently does not categorize L2 addresses
		// so we need to map them manually.
		// TODO: Update op-deployer to sort rollup contracts by category
		l2Addresses := make(DeploymentAddresses)
		for _, addressName := range []string{"optimismMintableERC20FactoryProxy"} {
			if addr, ok := l1Addresses[addressName]; ok {
				l2Addresses[addressName] = addr
				delete(l1Addresses, addressName)
			}
		}

		result.Deployments[id] = DeploymentState{
			L1Addresses: l1Addresses,
			L2Addresses: l2Addresses,
		}
	}

	result.Addresses = mapDeployment(state.ImplementationsDeployment)
	// merge the superchain and implementations addresses
	for key, value := range mapDeployment(state.SuperChainDeployment) {
		result.Addresses[key] = value
	}

	return result, nil
}

// ExtractData downloads and parses the op-deployer state
func (d *Deployer) ExtractData(ctx context.Context) (*DeployerData, error) {
	fs, err := ktfs.NewEnclaveFS(ctx, d.enclave)
	if err != nil {
		return nil, err
	}

	deployerArtifact, err := fs.GetArtifact(ctx, d.deployerArtifactName)
	if err != nil {
		return nil, err
	}

	stateBuffer := bytes.NewBuffer(nil)
	walletsBuffer := bytes.NewBuffer(nil)
	if err := deployerArtifact.ExtractFiles(
		ktfs.NewArtifactFileWriter(d.stateName, stateBuffer),
		ktfs.NewArtifactFileWriter(d.walletsName, walletsBuffer),
	); err != nil {
		return nil, err
	}

	state, err := parseStateFile(stateBuffer)
	if err != nil {
		return nil, err
	}

	l1WalletsForL2Admin, err := parseWalletsFile(walletsBuffer)
	if err != nil {
		return nil, err
	}

	// Generate test wallets from the standard "test test test..." mnemonic
	// These are the same wallets funded in L2Genesis.s.sol's devAccounts array
	devWallets, err := d.getDevWallets()
	if err != nil {
		return nil, err
	}

	for id, deployment := range state.Deployments {
		if l1Wallets, exists := l1WalletsForL2Admin[id]; exists {
			deployment.L1Wallets = l1Wallets
		}
		deployment.L2Wallets = devWallets

		genesisBuffer := bytes.NewBuffer(nil)
		genesisName, err := d.renderGenesisNameTemplate(id)
		if err != nil {
			return nil, err
		}

		if err := deployerArtifact.ExtractFiles(
			ktfs.NewArtifactFileWriter(genesisName, genesisBuffer),
		); err != nil {
			return nil, err
		}

		// Parse the genesis file JSON into a core.Genesis struct
		var genesis core.Genesis
		if err := json.NewDecoder(genesisBuffer).Decode(&genesis); err != nil {
			return nil, fmt.Errorf("failed to parse genesis file %s in artifact %s for chain ID %s: %w", genesisName, d.deployerArtifactName, id, err)
		}

		// Store the genesis data in the deployment state
		deployment.Config = genesis.Config
		state.Deployments[id] = deployment
	}

	l1GenesisArtifact, err := fs.GetArtifact(ctx, d.genesisArtifactName)
	if err != nil {
		return nil, err
	}

	l1ValidatorWallets, err := d.getL1ValidatorWallets(l1GenesisArtifact)
	if err != nil {
		return nil, err
	}

	l1ChainID, err := d.getL1ChainID(l1GenesisArtifact)
	if err != nil {
		return nil, err
	}

	return &DeployerData{
		L1ChainID:          l1ChainID,
		State:              state,
		L1ValidatorWallets: l1ValidatorWallets,
	}, nil
}

func (d *Deployer) renderGenesisNameTemplate(chainID string) (string, error) {
	tmpl, err := template.New("genesis").Parse(d.l2GenesisNameTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to compile genesis name template %s: %w", d.l2GenesisNameTemplate, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]string{"ChainID": chainID})
	if err != nil {
		return "", fmt.Errorf("failed to execute name template %s: %w", d.l2GenesisNameTemplate, err)
	}

	return buf.String(), nil
}

// getDevWallets generates the set of test wallets used in L2Genesis.s.sol
// These wallets are derived from the standard test mnemonic
func (d *Deployer) getDevWallets() ([]*Wallet, error) {
	m, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to create mnemonic dev keys: %w", err)
	}

	// Generate 30 wallets to match L2Genesis.s.sol's devAccounts array
	testWallets := make([]*Wallet, 0, 30)
	for i := 0; i < 30; i++ {
		key := devkeys.UserKey(uint64(i))
		addr, err := m.Address(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get address for test wallet %d: %w", i, err)
		}

		sec, err := m.Secret(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret key for test wallet %d: %w", i, err)
		}

		testWallets = append(testWallets, &Wallet{
			Name:       fmt.Sprintf("dev-account-%d", i),
			Address:    addr,
			PrivateKey: hexutil.Bytes(crypto.FromECDSA(sec)).String(),
		})
	}

	return testWallets, nil
}
