// Package v4_1_0 implements the upgrade to v4.1.0 (U16a). The interface for the upgrade is identical
// to the upgrade for v2.0.0 (U13), so all this package does is implement the Upgrader interface and
// call into the v2.0.0 upgrade.
package v4_1_0

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
	return artifacts.CreateHttpLocator("579f43b5bbb43e74216b7ed33125280567df86eaf00f7621f354e4a68c07323e")
}

var DefaultUpgrader = new(Upgrader)
