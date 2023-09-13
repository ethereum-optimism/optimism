package node

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/params"
)

func (n *OpNode) handleProtocolVersionsUpdate(ctx context.Context) {
	recommended := n.runCfg.RecommendedProtocolVersion()
	required := n.runCfg.RequiredProtocolVersion()
	// if the protocol version sources are disabled we do not process them
	if recommended == (params.ProtocolVersion{}) && required == (params.ProtocolVersion{}) {
		return
	}
	local := rollup.OPStackSupport
	// forward to execution engine, and get back the protocol version that op-geth supports
	engineSupport, err := n.l2Source.SignalSuperchainV1(ctx, recommended, required)
	if err != nil {
		n.log.Warn("failed to notify engine of protocol version", "err", err)
		// engineSupport may still be available, or otherwise zero to signal as unknown
	} else {
		catalyst.LogProtocolVersionSupport(n.log.New("node", "op-node"), engineSupport, recommended, "recommended")
		catalyst.LogProtocolVersionSupport(n.log.New("node", "op-node"), engineSupport, required, "required")
	}
	n.metrics.ReportProtocolVersions(local, engineSupport, recommended, required)
	catalyst.LogProtocolVersionSupport(n.log.New("node", "engine"), local, recommended, "recommended")
	catalyst.LogProtocolVersionSupport(n.log.New("node", "engine"), local, required, "required")

	// We may need to halt the node, if the user opted in to handling incompatible protocol-version signals
	n.HaltMaybe()
}

// HaltMaybe halts the rollup node if the runtime config indicates an incompatible required protocol change
// and the node is configured to opt-in to halting at this protocol-change level.
func (n *OpNode) HaltMaybe() {
	var needLevel int
	switch n.rollupHalt {
	case "major":
		needLevel = 3
	case "minor":
		needLevel = 2
	case "patch":
		needLevel = 1
	default:
		return // do not consider halting if not configured to
	}
	haveLevel := 0
	local := rollup.OPStackSupport
	required := n.runCfg.RequiredProtocolVersion()
	switch local.Compare(required) {
	case params.OutdatedMajor:
		haveLevel = 3
	case params.OutdatedMinor:
		haveLevel = 2
	case params.OutdatedPatch:
		haveLevel = 1
	}
	if haveLevel >= needLevel { // halt if we opted in to do so at this granularity
		n.log.Error("opted to halt, unprepared for protocol change", "required", required, "local", local)
		if err := n.Close(); err != nil {
			n.log.Error("failed to halt rollup", "err", err)
		}
	}
}
