// Package v4_0_0 implements the upgrade to v4.0.0 (U16). The interface for the upgrade is identical
// to the upgrade for v2.0.0 (U13), so all this package does is implement the Upgrader interface and
// call into the v2.0.0 upgrade.
package v4_0_0

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	v200 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/v2_0_0"
)

type Upgrader struct {
}

func (u *Upgrader) Upgrade(host *script.Host, input json.RawMessage) error {
	return v200.DefaultUpgrader.Upgrade(host, input)
}

func (u *Upgrader) ArtifactsURL() string {
	return artifacts.CreateHttpLocator("67966a2cb9945e1d9ab40e9c61f499e73cdb31d21b8d29a5a5c909b2b13ecd70")
}

var DefaultUpgrader = new(Upgrader)
