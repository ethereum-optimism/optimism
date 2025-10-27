package embedded

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	v600 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v6_0_0"
)

type Upgrader struct {
}

func (u *Upgrader) Upgrade(host *script.Host, input json.RawMessage) error {
	return v600.DefaultUpgrader.Upgrade(host, input)
}

func (u *Upgrader) ArtifactsURL() string {
	return artifacts.EmbeddedLocatorString
}

var DefaultUpgrader = new(Upgrader)
