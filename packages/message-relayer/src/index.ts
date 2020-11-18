import { JsonRpcProvider } from '@ethersproject/providers'
import { Contract, ethers, Wallet, BigNumber } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'
const rlp = require('rlp')

const TEST_OFFSET = 8

interface MessageRelayerOptions {
  l1RpcProvider: JsonRpcProvider
  l2RpcProvider: JsonRpcProvider
  stateCommitmentChainAddress: string
  l1CrossDomainMessengerAddress: string
  l2CrossDomainMessengerAddress: string
  l2ToL1MessagePasserAddress: string
  l2ChainStartingHeight: number
  pollingInterval: number
  relaySigner: Wallet
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
      let proof = await getMessageProof(l2RpcProvider, l2CrossDomainMessenger.address, options.l2ToL1MessagePasserAddress, stateCommitmentChain, message)

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
  l2CrossDomainMessengerAddress: string,
  l2ToL1MessagePasserAddress: string,
  stateCommitmentChain: Contract,
  message: any
): Promise<any> => {
  const messageHash = ethers.utils.keccak256(
    message.xDomainCalldata + l2CrossDomainMessengerAddress.slice(2)
  )

  const messageSlot = ethers.utils.keccak256(
    messageHash + '00'.repeat(32)
  )

  const proof = await l2RpcProvider.send('eth_getProof', [l2ToL1MessagePasserAddress, [messageSlot]])
  const batch = await getBatchHeader(stateCommitmentChain, message.index)

  console.log(proof.storageProof[0])

  return {
    stateRoot: proof.stateRoot,
    stateRootBatchHeader: batch,
    stateRootProof: {
      index: 0,
      siblings: [],
    },
    stateTrieWitness: rlp.encode(proof.accountProof),
    storageTrieWitness: rlp.encode(proof.storageProof[0].proof),
  }
}

const relayMessageToL1 = async (
  l1CrossDomainMessenger: Contract,
  message: any,
  proof: any,
  relaySigner: Wallet
): Promise<void> => {
  const transaction = await l1CrossDomainMessenger.populateTransaction.relayMessage(
    message.target,
    message.sender,
    message.data,
    message.nonce,
    proof
  )

  transaction.gasLimit = BigNumber.from(1000000)
  transaction.gasPrice = BigNumber.from(0)

  const signed = await relaySigner.signTransaction(transaction)

  await relaySigner.provider.sendTransaction(signed)
}

const getSentMessages = async (
  l2CrossDomainMessenger: Contract,
  startHeight: number,
  endHeight: number,
): Promise<any[]> => {
  const filter = l2CrossDomainMessenger.filters.SentMessage()
  const events = await l2CrossDomainMessenger.queryFilter(filter, startHeight + TEST_OFFSET, endHeight + TEST_OFFSET)
  
  return events.map((event) => {
    return parseMessageEvent(event)
  })
}

const getBatchHeader = async (
  stateCommitmentChain: Contract,
  transactionHeight: number,
): Promise<any> => {
  const filter = stateCommitmentChain.filters.StateBatchAppended(transactionHeight)
  const events = await stateCommitmentChain.queryFilter(filter)
  
  if (events.length === 0) {
    return
  }

  const args = events[0].args
  const batch = {
    batchIndex: args._batchIndex,
    batchRoot: args._batchRoot,
    batchSize: args._batchSize,
    prevTotalElements: args._prevTotalElements,
    extraData: args._extraData,
  }

  return batch
}

const isTransactionFinalized = async (
  stateCommitmentChain: Contract,
  transactionHeight: number,
): Promise<boolean> => {
  const batch = await getBatchHeader(stateCommitmentChain, transactionHeight)
  if (!batch) {
    return false
  }
  return !(await stateCommitmentChain.insideFraudProofWindow(batch))
}

const parseMessageEvent = (event: any): any => {
  const message = event.args.message
  const encoded = message.slice(10)
  const parts = ethers.utils.defaultAbiCoder.decode(
    ['address','address','bytes','uint256'],
    '0x' + encoded
  )

  return {
    target: parts[0],
    sender: parts[1],
    data: parts[2],
    nonce: parts[3],
    xDomainCalldata: message,
    hash: ethers.utils.keccak256(message),
    index: event.blockNumber - TEST_OFFSET
  }
}

const sleep = async (ms: number): Promise<void> => {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, ms)
  })
}
