package utils

import (
	"math/big"

	actionhelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"
)

var ProxyImplGetterFunc = w3.MustNewFunc(`implementation()`, `address`)

func RequireContractDeployedAndProxyUpdated(t actionhelpers.Testing, chain *dsl.Chain, implAddr common.Address, proxyAddress common.Address, activationBlockID eth.BlockID) {
	code, err := chain.SequencerEngine.EthClient().CodeAt(t.Ctx(), implAddr, big.NewInt(int64(activationBlockID.Number)))
	require.NoError(t, err)
	require.NotEmpty(t, code, "contract should be deployed")
	selector, err := ProxyImplGetterFunc.EncodeArgs()
	require.NoError(t, err)
	implAddrBytes, err := chain.SequencerEngine.EthClient().CallContract(t.Ctx(), ethereum.CallMsg{
		To:   &proxyAddress,
		Data: selector,
	}, big.NewInt(int64(activationBlockID.Number)))
	require.NoError(t, err)
	var implAddrActual common.Address
	err = ProxyImplGetterFunc.DecodeReturns(implAddrBytes, &implAddrActual)
	require.NoError(t, err)
	require.Equal(t, implAddr, implAddrActual)
}
