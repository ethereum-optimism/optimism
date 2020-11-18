import { JsonRpcProvider } from '@ethersproject/providers'
import { Contract, Signer } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'

interface MessageRelayerOptions {
  l1RpcProvider: JsonRpcProvider
  l2RpcProvider: JsonRpcProvider
  stateCommitmentChainAddress: string
  l1CrossDomainMessengerAddress: string
  l2CrossDomainMessengerAddress: string
  l2ChainStartingHeight: number
  pollingInterval: number
  relaySigner: Signer
}

export const main = async (options: MessageRelayerOptions) => {
  // Setup relevant objects.
  const l1RpcProvider = options.l1RpcProvider
  const l2RpcProvider = options.l2RpcProvider
  const stateCommitmentChain = new Contract(
    options.stateCommitmentChainAddress,
    getContractInterface('OVM_StateCommitmentChain'),
    l1RpcProvider
  )
  const l1CrossDomainMessenger = new Contract(
    options.l1CrossDomainMessengerAddress,
    getContractInterface('OVM_L1CrossDomainMessenger'),
    l1RpcProvider
  )
  const l2CrossDomainMessenger = new Contract(
    options.l2CrossDomainMessengerAddress,
    getContractInterface('OVM_L2CrossDomainMessenger'),
    l2RpcProvider
  )
  const relaySigner = options.relaySigner

  // Sanity checks.
  // TODO
  // Check that the Layer 1 provider is live.
  // Check that the Layer 2 provider is live.
  // Check that the StateCommitmentChain is valid.
  // Check that the L1CrossDomainMessenger is valid.
  // Check that the L2CrossDomainMessenger is valid.
  // Check the starting height.
  // Check the polling interval.

  // Variable setup.
  let currentFinalizedTransactionHeight = options.l2ChainStartingHeight
  let nextUnfinalizedTransactionHeight = options.l2ChainStartingHeight + 1

  // Primary loop.
  while (true) {
    await sleep(options.pollingInterval)

    // Check that transaction N has been finalized.
    if (
      !(await isTransactionFinalized(
        stateCommitmentChain,
        nextUnfinalizedTransactionHeight
      ))
    ) {
      continue
    } else {
      currentFinalizedTransactionHeight = nextUnfinalizedTransactionHeight
      while (
        await isTransactionFinalized(
          stateCommitmentChain,
          nextUnfinalizedTransactionHeight
        )
      ) {
        nextUnfinalizedTransactionHeight++
      }
    }

    // Find all sent message events on Layer 2 within the range.
    const messages = await getSentMessages(
      l2CrossDomainMessenger,
      currentFinalizedTransactionHeight,
      nextUnfinalizedTransactionHeight
    )

    for (const message of messages) {
      // Check L1CrossDomainMessenger that the message has not been relayed
      if (await wasMessageRelayed(l1CrossDomainMessenger, message)) {
        continue
      }

      // Get proof for the message from Layer 2.
      let proof = await getMessageProof(l2RpcProvider, message)

      // Send the message and proof to the L1CrossDomainMessenger.
      await relayMessageToL1(
        l1CrossDomainMessenger,
        message,
        proof,
        relaySigner
      )
    }
  }
}

const wasMessageRelayed = async (
  l1CrossDomainMessenger: Contract,
  message: any
): Promise<boolean> => {
  return l1CrossDomainMessenger.successfulMessages(message.hash)
}

const getMessageProof = async (
  l2RpcProvider: JsonRpcProvider,
  message: any
): Promise<any> => {
  return false
}

const relayMessageToL1 = async (
  l1CrossDomainMessenger: Contract,
  message: any,
  proof: any,
  relaySigner: Signer
): Promise<void> => {
  return l1CrossDomainMessenger.relayMessage(
    message.target,
    message.sender,
    message.data,
    message.nonce,
    proof,
    { from: relaySigner, gasLimit: message.gasLimit }
  )
}

//const getSentMessages =

//const isTransactionFinalized =

const sleep = async (ms: number): Promise<void> => {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, ms)
  })
}
