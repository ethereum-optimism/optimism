// Package v5_0_0 implements the upgrade to v5.0.0 (U17). The interface for the upgrade is identical
// to the upgrade for v2.0.0 (U13), so all this package does is implement the Upgrader interface and
// call into the v2.0.0 upgrade.
package v5_0_0

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	v200 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v2_0_0"
)

type Upgrader struct{}

func (u *Upgrader) Upgrade(host *script.Host, input json.RawMessage) error {
	return v200.DefaultUpgrader.Upgrade(host, input)
}

func (u *Upgrader) ArtifactsURL() string {
	return artifacts.CreateHttpLocator("b112b16f8939fbb732c0693de3d3bd1e8e3e2f0771f91d5ab300a6c9b7b1af73")
}

var DefaultUpgrader = new(Upgrader)
