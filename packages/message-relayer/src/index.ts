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
  pollingInterval: number  // milliseconds
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

  let currentFinalizedTransactionHeight = options.l2ChainStartingHeight
  let nextUnfinalizedTransactionHeight = options.l2ChainStartingHeight+1
  while(true) {

    await sleep(pollingInterval)
    // Check that transaction N has been finalized.
    if ( !(await isTransactionFinalized(stateCommitmentChain, nextUnfinalizedTransactionHeight)) ) {
      continue
    } else {
      currentFinalizedTransactionHeight = nextUnfinalizedTransactionHeight
      while(await isTransactionFinalized(stateCommitmentChain, nextUnfinalizedTransactionHeight)) {
        nextUnfinalizedTransactionHeight++
      }
    }

    // Find all sent message events on Layer 2 within the range.
    const messages = await getSentMessages(l2CrossDomainMessenger, currentFinalizedTransactionHeight, nextUnfinalizedTransactionHeight)

    for (const message of messages) {

      //Check L1CrossDomainMessenger that the message has not been relayed
      if ( await wasMessageRelayed(l1CrossDomainMessenger, message) ) {
        continue
      }
      // Get proof for the message from Layer 2.
      let proof = await getMessageProof(l2RpcProvider, message)
          // Send the message and proof to the L1CrossDomainMessenger.
          await relayMessageToL1(l1CrossDomainMessenger, message, proof, relayerWallet)

    }

  }
}

const wasMessageRelayed = async (l1CrossDomainMessenger:Contract, message:any): Promise<boolean> => {
  return (l1CrossDomainMessenger.successfulMessages(message.hash))
}

const getMessageProof = async(l2RpcProvider:JsonRpcProvider, message:any): Promise<any> => {
  return false
}

const relayMessageToL1 = async(l1CrossDomainMessenger:Contract, message:any, proof:any, relayerWallet:Wallet): Promise<void> => {
  return (l1CrossDomainMessenger.relayMessage(message.target, message.sender, message.data, message.nonce, proof, {from:relayerWallet, gasLimit:message.gasLimit}))
}

//const getSentMessages =

//const isTransactionFinalized =



const sleep = async (ms: number): Promise<void> => {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, ms)
  })
}