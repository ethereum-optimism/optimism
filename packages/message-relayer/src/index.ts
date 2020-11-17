import { JsonRpcProvider } from '@ethersproject/providers'
import { Contract, Wallet } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'

interface MessageRelayerOptions {
  l1RpcEndpoint: string
  l2RpcEndpoint: string
  stateCommitmentChainAddress: string
  l1CrossDomainMessengerAddress: string
  l2CrossDomainMessengerAddress: string
  l2ChainStartingHeight: number
  pollingInterval: number
  relayerPrivateKey: string
}

export const main = async (options: MessageRelayerOptions) => {
  // Setup relevant objects.

  // Layer 1 provider.
  // Layer 2 provider.
  // StateCommitmentChain contract.
  // L1CrossDomainMessenger contract.
  // L2CrossDomainMessenger contract.
  // Relaying account from private key.

  const l1RpcProvider = new JsonRpcProvider(options.l1RpcEndpoint)
  const l2RpcProvider = new JsonRpcProvider(options.l2RpcEndpoint)
  const stateCommitmentChain = new Contract(options.stateCommitmentChainAddress, getContractInterface('OVM_StateCommitmentChain'), l1RpcProvider)
  const l1CrossDomainMessenger = new Contract(options.l1CrossDomainMessengerAddress, getContractInterface('OVM_L1CrossDomainMessenger'), l1RpcProvider)
  const l2CrossDomainMessenger = new Contract(options.l2CrossDomainMessengerAddress, getContractInterface('OVM_L2CrossDomainMessenger'), l2RpcProvider)
  const relayerWallet = new Wallet(options.relayerPrivateKey, l1RpcProvider)


  // Sanity checks.

  // Check that the Layer 1 provider is live.
  // Check that the Layer 2 provider is live.
  // Check that the StateCommitmentChain is valid.
  // Check that the L1CrossDomainMessenger is valid.
  // Check that the L2CrossDomainMessenger is valid.
  // Check the starting height.
  // Check the polling interval.


  // Primary loop.

  let nextUnfinalizedTransactionHeight = options.l2ChainStartingHeight

  // Check that transaction N has been finalized.
    // If no:
      // Do nothing, wait until next loop.
    // If yes:
      // Iteratively find the next unfinalized transaction (becomes transaction N).
      // Gives us a range of newly finalized transactions.
  // Find all sent message events on Layer 2 within the range.
    // For each event:
      // Check that the message has not been relayed (call the L1CrossDomainMessenger).
      // Get proof for the message from Layer 2.
      // Send the message and proof to the L1CrossDomainMessenger.
}
