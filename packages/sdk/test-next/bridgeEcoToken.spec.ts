import ethers from 'ethers'
import { describe, expect, it } from 'vitest'
import {
  l1TestClient,
  l2TestClient,
  l1PublicClient,
  l2PublicClient,
} from './testUtils/viemClients'

import { BRIDGE_ADAPTER_DATA, CrossChainMessenger, DEPOSIT_CONFIRMATION_BLOCKS, L2ChainID } from '../src'
import { Address, PublicClient, parseEther } from 'viem'
import { l1Provider, l2Provider } from './testUtils/ethersProviders'

const ECO_WHALE: Address = '0xBd11c836279a1352ce737FbBFba36b20734B04e7'

// TODO: use tokenlist as source of truth
const ECO_L1_TOKEN_ADDRESS: Address = '0x3E87d4d9E69163E7590f9b39a70853cf25e5ABE3'
const ECO_L2_TOKEN_ADDRESS: Address = '0x54bBECeA38ff36D32323f8A754683C1F5433A89f'

const getERC20TokenBalance = async (publicClient: PublicClient, tokenAddress: Address, ownerAddress: Address) => {
  const result = await publicClient.readContract({
    address: tokenAddress,
    abi: [
      {
        inputs: [{ name: 'owner', type: 'address' }],
        name: 'balanceOf',
        outputs: [{ name: '', type: 'uint256' }],
        stateMutability: 'view',
        type: 'function',
      }
    ],
    functionName: 'balanceOf',
    args: [ownerAddress]
  })

  return result as bigint
}

const getL1ERC20TokenBalance = async (ownerAddress: Address) => {
  return getERC20TokenBalance(l1PublicClient, ECO_L1_TOKEN_ADDRESS, ownerAddress)
}

const getL2ERC20TokenBalance = async (ownerAddress: Address) => {
  return getERC20TokenBalance(l2PublicClient, ECO_L2_TOKEN_ADDRESS, ownerAddress)
}

describe('ECO token', () => {
  it('sdk should be able to deposit to l1 bridge contract correctly', async () => {
    // this code is a bit whack because of the mix of viem + ethers
    // TODO: use anviljs, and use viem entirely once the sdk supports it
    await l1TestClient.impersonateAccount({ address: ECO_WHALE })
    const l1EcoWhaleSigner = await l1Provider.getSigner(ECO_WHALE);

    const preBridgeL1EcoWhaleBalance = await getL1ERC20TokenBalance(ECO_WHALE)

    const crossChainMessenger = new CrossChainMessenger({
      l1SignerOrProvider: l1EcoWhaleSigner,
      l2SignerOrProvider: l2Provider,
      l1ChainId: 5,
      l2ChainId: 420,
      bedrock: true,
      bridges: BRIDGE_ADAPTER_DATA[L2ChainID.OPTIMISM_GOERLI]
    })

    await crossChainMessenger.approveERC20(
      ECO_L1_TOKEN_ADDRESS,
      ECO_L2_TOKEN_ADDRESS,
      ethers.utils.parseEther('0.1'),
    )

    const txResponse = await crossChainMessenger.depositERC20(
      ECO_L1_TOKEN_ADDRESS,
      ECO_L2_TOKEN_ADDRESS,
      ethers.utils.parseEther('0.1'),
    )

    await txResponse.wait();

    const l1EcoWhaleBalance = await getL1ERC20TokenBalance(ECO_WHALE)
    expect(l1EcoWhaleBalance).toEqual(preBridgeL1EcoWhaleBalance - parseEther('0.1'))
  }, 20_000)

  it('sdk should be able to withdraw into the l2 bridge contract correctly', async () => {
    await l2TestClient.impersonateAccount({ address: ECO_WHALE })
    const l2EcoWhaleSigner = await l2Provider.getSigner(ECO_WHALE);

    const preBridgeL2EcoWhaleBalance = await getL2ERC20TokenBalance(ECO_WHALE)

    const crossChainMessenger = new CrossChainMessenger({
      l1SignerOrProvider: l1Provider,
      l2SignerOrProvider: l2EcoWhaleSigner,
      l1ChainId: 5,
      l2ChainId: 420,
      bedrock: true,
      bridges: BRIDGE_ADAPTER_DATA[L2ChainID.OPTIMISM_GOERLI]
    })

    const txResponse = await crossChainMessenger.withdrawERC20(
      ECO_L1_TOKEN_ADDRESS,
      ECO_L2_TOKEN_ADDRESS,
      ethers.utils.parseEther('0.1'),
    )

    await txResponse.wait();

    const l2EcoWhaleBalance = await getL2ERC20TokenBalance(ECO_WHALE)
    expect(l2EcoWhaleBalance).toEqual(preBridgeL2EcoWhaleBalance - parseEther('0.1'))
  }, 20_000)

})
