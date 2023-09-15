package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

type ethereumContracts struct {
	AddressManager                    string `json:"AddressManager"`
	BlockOracle                       string `json:"BlockOracle"`
	DisputeGameFactory                string `json:"DisputeGameFactory"`
	DisputeGameFactoryProxy           string `json:"DisputeGameFactoryProxy"`
	L1CrossDomainMessenger            string `json:"L1CrossDomainMessenger"`
	L1CrossDomainMessengerProxy       string `json:"L1CrossDomainMessengerProxy"`
	L1ERC721Bridge                    string `json:"L1ERC721Bridge"`
	L1ERC721BridgeProxy               string `json:"L1ERC721BridgeProxy"`
	L1StandardBridge                  string `json:"L1StandardBridge"`
	L1StandardBridgeProxy             string `json:"L1StandardBridgeProxy"`
	L2OutputOracle                    string `json:"L2OutputOracle"`
	L2OutputOracleProxy               string `json:"L2OutputOracleProxy"`
	Mips                              string `json:"Mips"`
	OptimismMintableERC20Factory      string `json:"OptimismMintableERC20Factory"`
	OptimismMintableERC20FactoryProxy string `json:"OptimismMintableERC20FactoryProxy"`
	OptimismPortal                    string `json:"OptimismPortal"`
	OptimismPortalProxy               string `json:"OptimismPortalProxy"`
	PreimageOracle                    string `json:"PreimageOracle"`
	ProxyAdmin                        string `json:"ProxyAdmin"`
	SystemConfig                      string `json:"SystemConfig"`
	SystemConfigProxy                 string `json:"SystemConfigProxy"`
}

func readEthereumContracts(filename string) (*ethereumContracts, error) {
	// Check if the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}

	// Read the file
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON
	var contracts ethereumContracts
	if err := json.Unmarshal(content, &contracts); err != nil {
		return nil, err
	}

	return &contracts, nil
}

func GetDevnetPreset() (*Preset, error) {
	ethereumContracts, err := readEthereumContracts("../.devnet/addresses.json")
	if err != nil {
		return nil, err
	}
	return &Preset{
		Name: "devnet",
		ChainConfig: ChainConfig{
			L1StartingHeight:        0,
			L1BedrockStartingHeight: 0,
			L2BedrockStartingHeight: 0,
			L1Contracts: L1Contracts{
				AddressManager:                  common.HexToAddress(ethereumContracts.AddressManager),
				SystemConfigProxy:               common.HexToAddress(ethereumContracts.SystemConfigProxy),
				OptimismPortalProxy:             common.HexToAddress(ethereumContracts.OptimismPortalProxy),
				L2OutputOracleProxy:             common.HexToAddress(ethereumContracts.L2OutputOracleProxy),
				L1CrossDomainMessengerProxy:     common.HexToAddress(ethereumContracts.L1CrossDomainMessengerProxy),
				L1StandardBridgeProxy:           common.HexToAddress(ethereumContracts.L1StandardBridgeProxy),
				L1ERC721BridgeProxy:             common.HexToAddress(ethereumContracts.L1ERC721BridgeProxy),
				LegacyCanonicalTransactionChain: common.HexToAddress("0x0"),
				LegacyStateCommitmentChain:      common.HexToAddress("0x0"),
			},
		},
	}, nil
}
