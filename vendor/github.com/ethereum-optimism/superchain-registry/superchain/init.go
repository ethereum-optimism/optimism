package superchain

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	var err error
	SuperchainSemver = make(map[string]ContractVersions)

	superchainTargets, err := superchainFS.ReadDir("configs")
	if err != nil {
		panic(fmt.Errorf("failed to read superchain dir: %w", err))
	}
	// iterate over superchain-target entries
	for _, s := range superchainTargets {

		if !s.IsDir() {
			continue // ignore files, e.g. a readme
		}

		SuperchainSemver[s.Name()], err = newContractVersions(s.Name())
		if err != nil {
			panic(fmt.Errorf("failed to read semver.yaml: %w", err))
		}

		// Load superchain-target config
		superchainConfigData, err := superchainFS.ReadFile(path.Join("configs", s.Name(), "superchain.yaml"))
		if err != nil {
			panic(fmt.Errorf("failed to read superchain config: %w", err))
		}
		var superchainEntry Superchain
		if err := unMarshalSuperchainConfig(superchainConfigData, &superchainEntry.Config); err != nil {
			panic(fmt.Errorf("failed to decode superchain config: %w", err))
		}
		superchainEntry.Superchain = s.Name()

		// iterate over the chains of this superchain-target
		chainEntries, err := superchainFS.ReadDir(path.Join("configs", s.Name()))
		if err != nil {
			panic(fmt.Errorf("failed to read superchain dir: %w", err))
		}
		for _, c := range chainEntries {
			if !isConfigFile(c) {
				continue
			}
			// load chain config
			chainConfigData, err := superchainFS.ReadFile(path.Join("configs", s.Name(), c.Name()))
			if err != nil {
				panic(fmt.Errorf("failed to read superchain config %s/%s: %w", s.Name(), c.Name(), err))
			}
			var chainConfig ChainConfig

			if err := yaml.Unmarshal(chainConfigData, &chainConfig); err != nil {
				panic(fmt.Errorf("failed to decode chain config %s/%s: %w", s.Name(), c.Name(), err))
			}
			chainConfig.Chain = strings.TrimSuffix(c.Name(), ".yaml")

			(&chainConfig).setNilHardforkTimestampsToDefault(&superchainEntry.Config)

			MustBeValidSuperchainLevel(chainConfig)

			jsonName := chainConfig.Chain + ".json"
			addressesData, err := extraFS.ReadFile(path.Join("extra", "addresses", s.Name(), jsonName))
			if err != nil {
				panic(fmt.Errorf("failed to read addresses data of chain %s/%s: %w", s.Name(), jsonName, err))
			}
			var addrs AddressList
			if err := json.Unmarshal(addressesData, &addrs); err != nil {
				panic(fmt.Errorf("failed to decode addresses %s/%s: %w", s.Name(), jsonName, err))
			}

			genesisSysCfgData, err := extraFS.ReadFile(path.Join("extra", "genesis-system-configs", s.Name(), jsonName))
			if err != nil {
				panic(fmt.Errorf("failed to read genesis system config data of chain %s/%s: %w", s.Name(), jsonName, err))
			}
			var genesisSysCfg GenesisSystemConfig
			if err := json.Unmarshal(genesisSysCfgData, &genesisSysCfg); err != nil {
				panic(fmt.Errorf("failed to decode genesis system config %s/%s: %w", s.Name(), jsonName, err))
			}

			chainConfig.Superchain = s.Name()
			if other, ok := OPChains[chainConfig.ChainID]; ok {
				panic(fmt.Errorf("found chain config %q in superchain target %q with chain ID %d "+
					"conflicts with chain %q in superchain %q and chain ID %d",
					chainConfig.Name, chainConfig.Superchain, chainConfig.ChainID,
					other.Name, other.Superchain, other.ChainID))
			}
			superchainEntry.ChainIDs = append(superchainEntry.ChainIDs, chainConfig.ChainID)
			OPChains[chainConfig.ChainID] = &chainConfig
			Addresses[chainConfig.ChainID] = &addrs
			GenesisSystemConfigs[chainConfig.ChainID] = &genesisSysCfg

		}

		ciMainnetRPC := os.Getenv("CIRCLE_CI_MAINNET_RPC")
		ciSepoliaRPC := os.Getenv("CIRCLE_CI_SEPOLIA_RPC")

		switch superchainEntry.Superchain {
		case "mainnet":
			if ciMainnetRPC != "" {
				fmt.Println("Using env var for mainnet rpc")
				superchainEntry.Config.L1.PublicRPC = ciMainnetRPC
			}
		case "sepolia", "sepolia-dev-0":
			if ciSepoliaRPC != "" {
				fmt.Println("Using env var for sepolia rpc")
				superchainEntry.Config.L1.PublicRPC = ciSepoliaRPC
			}
		}

		Superchains[superchainEntry.Superchain] = &superchainEntry

		implementations, err := newContractImplementations(s.Name())
		if err != nil {
			panic(fmt.Errorf("failed to read implementations of superchain target %s: %w", s.Name(), err))
		}

		Implementations[s.Name()] = implementations
	}
}

func MustBeValidSuperchainLevel(chainConfig ChainConfig) {
	if chainConfig.SuperchainLevel != Frontier && chainConfig.SuperchainLevel != Standard {
		panic(fmt.Sprintf("invalid or unspecified superchain level %d", chainConfig.SuperchainLevel))
	}
}
