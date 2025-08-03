package deployer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	ktfs "github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/yaml.v3"
)

const (
	// Default number of L1 validator wallets if count is not specified in YAML
	defaultNumWallets = 21
)

// mnemonicConfig represents the structure of a mnemonic configuration entry
type mnemonicConfig struct {
	Mnemonic string `yaml:"mnemonic"`
	Count    int    `yaml:"count"`
}

// mnemonicResult contains the parsed mnemonic and count
type mnemonicResult struct {
	Mnemonic string
	Count    int
}

func getMnemonics(r io.Reader) (*mnemonicResult, error) {
	var config []mnemonicConfig
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode mnemonic config: %w", err)
	}

	if len(config) == 0 {
		return nil, fmt.Errorf("no mnemonic configurations found")
	}

	// Use the first mnemonic configuration
	// TODO: what does this mean if there are multiple mnemonics in this file?
	result := &mnemonicResult{
		Mnemonic: config[0].Mnemonic,
		Count:    config[0].Count,
	}

	// Validate the mnemonic is not empty
	if result.Mnemonic == "" {
		return nil, fmt.Errorf("mnemonic cannot be empty")
	}

	// Use default count if not specified or invalid
	if result.Count <= 0 {
		result.Count = defaultNumWallets
	}

	return result, nil
}

func (d *Deployer) getL1ValidatorWallets(deployerArtifact *ktfs.Artifact) ([]*Wallet, error) {
	mnemonicsBuffer := bytes.NewBuffer(nil)
	if err := deployerArtifact.ExtractFiles(
		ktfs.NewArtifactFileWriter(d.l1ValidatorMnemonicName, mnemonicsBuffer),
	); err != nil {
		return nil, err
	}

	mnemonicResult, err := getMnemonics(mnemonicsBuffer)
	if err != nil {
		return nil, err
	}

	m, err := devkeys.NewMnemonicDevKeys(mnemonicResult.Mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to create mnemonic dev keys: %w", err)
	}

	knownWallets := make([]*Wallet, 0, mnemonicResult.Count)

	var keys []devkeys.Key
	for i := 0; i < mnemonicResult.Count; i++ {
		keys = append(keys, devkeys.UserKey(i))
	}

	for _, key := range keys {
		addr, err := m.Address(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get address for key %s: %w", key.String(), err)
		}

		sec, err := m.Secret(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret for key %s: %w", key.String(), err)
		}

		knownWallets = append(knownWallets, &Wallet{
			Name:       key.String(),
			Address:    addr,
			PrivateKey: hexutil.Bytes(crypto.FromECDSA(sec)).String(),
		})
	}

	return knownWallets, nil
}

func (d *Deployer) getConfig(genesisArtifact *ktfs.Artifact) (*params.ChainConfig, error) {
	genesisBuffer := bytes.NewBuffer(nil)
	if err := genesisArtifact.ExtractFiles(
		ktfs.NewArtifactFileWriter(d.l1GenesisName, genesisBuffer),
	); err != nil {
		return nil, err
	}

	// Parse the genesis file JSON into a core.Genesis struct
	var genesis core.Genesis
	if err := json.NewDecoder(genesisBuffer).Decode(&genesis); err != nil {
		return nil, fmt.Errorf("failed to parse genesis file %s in artifact %s: %w", d.l1GenesisName, d.genesisArtifactName, err)
	}

	if genesis.Config == nil {
		return nil, fmt.Errorf("genesis config is nil in file %s", d.l1GenesisName)
	}

	return genesis.Config, nil
}
