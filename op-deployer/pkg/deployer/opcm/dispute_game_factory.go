package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

type SetDisputeGameImplInput struct {
	Factory             common.Address
	Impl                common.Address
	AnchorStateRegistry common.Address
	GameType            uint32
	GameArgs            []byte
}

func SetDisputeGameImpl(
	h *script.Host,
	input SetDisputeGameImplInput,
) error {
	return RunScriptVoid[SetDisputeGameImplInput](
		h,
		input,
		"SetDisputeGameImpl.s.sol",
		"SetDisputeGameImpl",
	)
}

func NewSetDisputeGameImplForgeCaller(client *forge.Client) forge.ScriptCaller[SetDisputeGameImplInput, forge.None] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/SetDisputeGameImpl.s.sol:SetDisputeGameImpl",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[SetDisputeGameImplInput]{TypeName: "SetDisputeGameImplInput"},
		forge.NoneDecoder{},
	)
}

// SetDisputeGameImplViaForge registers a dispute game implementation on its factory using Forge
func SetDisputeGameImplViaForge(env *ForgeEnv, input SetDisputeGameImplInput) error {
	if _, err := broadcastScript(env, NewSetDisputeGameImplForgeCaller, input); err != nil {
		return fmt.Errorf("failed to set the dispute game implementation with Forge: %w", err)
	}
	return nil
}
