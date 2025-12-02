package prestate

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/types"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/util"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/superchain"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const (
	monorepoGoModAtTag            = "https://github.com/ethereum-optimism/optimism/raw/refs/tags/%s/go.mod"
	superchainRegistryCommitAtRef = "https://github.com/ethereum-optimism/op-geth/raw/%s/superchain-registry-commit.txt"
	superchainConfigsZipAtRef     = "https://github.com/ethereum-optimism/op-geth/raw/%s/superchain/superchain-configs.zip"
)

type OPProgramPrestate struct {
}

func NewOPProgramPrestate() *OPProgramPrestate {
	return &OPProgramPrestate{}
}

func (p *OPProgramPrestate) FindVersions(log log.Logger, prestateVersion string) (
	elCommitInfo types.CommitInfo,
	fppCommitInfo types.CommitInfo,
	superChainRegistryCommit string,
	prestateConfigs *superchain.ChainConfigLoader,
) {
	prestateTag := fmt.Sprintf("op-program/v%s", prestateVersion)
	log.Info("Found prestate tag", "tag", prestateTag)
	fppCommitInfo = types.NewCommitInfo("ethereum-optimism", "optimism", prestateTag, "develop", "")

	modFile, err := fetchMonorepoGoMod(prestateTag)
	if err != nil {
		log.Crit("Failed to fetch go mod", "err", err)
	}
	elVersion := resolvePseudoVersion(p.findOpGethVersion(log, modFile))
	elCommitInfo = types.NewCommitInfo("ethereum-optimism", "op-geth", elVersion, "optimism", "")

	registryCommitBytes, err := util.Fetch(fmt.Sprintf(superchainRegistryCommitAtRef, elVersion))
	if err != nil {
		log.Crit("Failed to fetch superchain registry commit info", "err", err)
	}
	superChainRegistryCommit = strings.TrimSpace(string(registryCommitBytes))
	log.Info("Found superchain registry commit info", "commit", superChainRegistryCommit)

	prestateConfigData, err := util.Fetch(fmt.Sprintf(superchainConfigsZipAtRef, elVersion))
	if err != nil {
		log.Crit("Failed to fetch prestate's superchain registry config zip", "err", err)
	}
	configLoader, err := superchain.NewChainConfigLoader(prestateConfigData)
	if err != nil {
		log.Crit("Failed to parse prestate's superchain registry config zip", "err", err)
	}
	prestateConfigs = configLoader
	return
}

func (p *OPProgramPrestate) findOpGethVersion(log log.Logger, modFile *modfile.File) string {
	var elVersion string
	for _, replace := range modFile.Replace {
		if replace.Old.Path == "github.com/ethereum/go-ethereum" {
			elVersion = replace.New.Version
			break
		}
	}
	if elVersion == "" {
		log.Crit("Failed to find op-geth replace in go.mod")
	}
	log.Info("Found op-geth version", "version", elVersion)
	return elVersion
}

func fetchMonorepoGoMod(opProgramTag string) (*modfile.File, error) {
	goModUrl := fmt.Sprintf(monorepoGoModAtTag, opProgramTag)
	goMod, err := util.Fetch(goModUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch go.mod: %w", err)
	}

	return modfile.Parse("go.mod", goMod, nil)
}

// resolvePseudoVersion converts a Go module version to a git ref.
// For pseudo-versions like "v1.101604.0-synctest.0.0.20251208094937-ba6bdcfef423",
// it extracts the commit hash suffix. For regular tags, it returns the version as-is.
func resolvePseudoVersion(version string) string {
	if module.IsPseudoVersion(version) {
		rev, err := module.PseudoVersionRev(version)
		if err != nil {
			log.Crit("Failed to extract commit hash from pseudo-version", "version", version, "err", err)
		}
		return rev
	}
	return version
}
