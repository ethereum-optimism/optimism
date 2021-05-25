/* Imports: External */
import { ethers } from 'ethers'
import {
  fromHexString,
  remove0x,
  toHexString,
  toRpcHexString,
} from '@eth-optimism/core-utils'
import { getContractInterface } from '@eth-optimism/contracts'
import * as rlp from 'rlp'
import { MerkleTree } from 'merkletreejs'

// Number of blocks added to the L2 chain before the first L2 transaction. Genesis are added to the
// chain to initialize the system. However, they create a discrepancy between the L2 block number
// the index of the transaction that corresponds to that block number. For example, if there's 1
// genesis block, then the transaction with an index of 0 corresponds to the block with index 1.
const NUM_L2_GENESIS_BLOCKS = 1

interface StateRootBatchHeader {
  batchIndex: ethers.BigNumber
  batchRoot: string
  batchSize: ethers.BigNumber
  prevTotalElements: ethers.BigNumber
  extraData: string
}

interface StateRootBatch {
  header: StateRootBatchHeader
  stateRoots: string[]
}

interface CrossDomainMessage {
  target: string
  sender: string
  message: string
  messageNonce: number
}

interface StateTrieProof {
  accountProof: string
  storageProof: string
}

const getMessageByTransactionHash = async (
  l2RpcProvider: ethers.providers.JsonRpcProvider,
  l2CrossDomainMessengerAddress: string,
  l2TransactionHash: string
): Promise<CrossDomainMessage> => {
  const transaction = await l2RpcProvider.getTransaction(l2TransactionHash)
  if (transaction === null) {
    throw new Error(`unable to find tx with hash: ${l2TransactionHash}`)
  }

  const l2CrossDomainMessenger = new ethers.Contract(
    l2CrossDomainMessengerAddress,
    getContractInterface('OVM_L2CrossDomainMessenger'),
    l2RpcProvider
  )

  const sentMessageEvents = await l2CrossDomainMessenger.queryFilter(
    l2CrossDomainMessenger.filters.SentMessage(),
    transaction.blockNumber,
    transaction.blockNumber
  )

  if (sentMessageEvents.length === 0) {
    return null
  }

  if (sentMessageEvents.length > 1) {
    throw new Error(
      `can currently only support one message per transaction, found ${sentMessageEvents}`
    )
  }

  const encodedMessage = sentMessageEvents[0].args.message
  const decodedMessage = l2CrossDomainMessenger.interface.decodeFunctionData(
    'relayMessage',
    encodedMessage
  )

  return {
    target: decodedMessage._target,
    sender: decodedMessage._sender,
    message: decodedMessage._message,
    messageNonce: decodedMessage._messageNonce.toNumber(),
  }
}

const encodeCrossDomainMessage = (message: CrossDomainMessage): string => {
  return getContractInterface(
    'OVM_L2CrossDomainMessenger'
  ).encodeFunctionData('relayMessage', [
    message.target,
    message.sender,
    message.message,
    message.messageNonce,
  ])
}

export const getStateBatchAppendedEventByTransactionIndex = async (
  l1RpcProvider: ethers.providers.JsonRpcProvider,
  l1StateCommitmentChainAddress: string,
  l2TransactionIndex: number
): Promise<ethers.Event | null> => {
  const l1StateCommitmentChain = new ethers.Contract(
    l1StateCommitmentChainAddress,
    getContractInterface('OVM_StateCommitmentChain'),
    l1RpcProvider
  )

  const getStateBatchAppendedEventByBatchIndex = async (
    index: number
  ): Promise<ethers.Event | null> => {
    const eventQueryResult = await l1StateCommitmentChain.queryFilter(
      l1StateCommitmentChain.filters.StateBatchAppended(index)
    )
    if (eventQueryResult.length === 0) {
      return null
    } else {
      return eventQueryResult[0]
    }
  }

  const totalBatches = await l1StateCommitmentChain.getTotalBatches()
  if (totalBatches === 0) {
    return null
  }

  const isEventHi = (event: ethers.Event, index: number) => {
    const prevTotalElements = event.args._prevTotalElements.toNumber()
    return index < prevTotalElements
  }

  const isEventLo = (event: ethers.Event, index: number) => {
    const prevTotalElements = event.args._prevTotalElements.toNumber()
    const batchSize = event.args._batchSize.toNumber()
    return index >= prevTotalElements + batchSize
  }

  const lastBatchEvent = await getStateBatchAppendedEventByBatchIndex(
    totalBatches - 1
  )
  if (isEventLo(lastBatchEvent, l2TransactionIndex)) {
    return null
  } else if (!isEventHi(lastBatchEvent, l2TransactionIndex)) {
    return lastBatchEvent
  }

  let lowerBound = 0
  let upperBound = totalBatches - 1
  let batchEvent: ethers.Event | null = lastBatchEvent
  while (lowerBound < upperBound) {
    const middleOfBounds = Math.floor((lowerBound + upperBound) / 2)
    batchEvent = await getStateBatchAppendedEventByBatchIndex(middleOfBounds)

    if (isEventHi(batchEvent, l2TransactionIndex)) {
      upperBound = middleOfBounds
    } else if (isEventLo(batchEvent, l2TransactionIndex)) {
      lowerBound = middleOfBounds
    } else {
      break
    }
  }

  return batchEvent
}

export const getStateRootBatchByIndex = async (
  l1RpcProvider: ethers.providers.JsonRpcProvider,
  l1StateCommitmentChainAddress: string,
  l2TransactionIndex: number
): Promise<StateRootBatch | null> => {
  const l1StateCommitmentChain = new ethers.Contract(
    l1StateCommitmentChainAddress,
    getContractInterface('OVM_StateCommitmentChain'),
    l1RpcProvider
  )

  const stateBatchAppendedEvent = await getStateBatchAppendedEventByTransactionIndex(
    l1RpcProvider,
    l1StateCommitmentChainAddress,
    l2TransactionIndex
  )
  if (stateBatchAppendedEvent === null) {
    return null
  }

  const stateBatchTransaction = await stateBatchAppendedEvent.getTransaction()
  const [stateRoots] = l1StateCommitmentChain.interface.decodeFunctionData(
    'appendStateBatch',
    stateBatchTransaction.data
  )

  return {
    header: {
      batchIndex: stateBatchAppendedEvent.args._batchIndex,
      batchRoot: stateBatchAppendedEvent.args._batchRoot,
      batchSize: stateBatchAppendedEvent.args._batchSize,
      prevTotalElements: stateBatchAppendedEvent.args._prevTotalElements,
      extraData: stateBatchAppendedEvent.args._extraData,
    },
    stateRoots,
  }
}

const getMerkleTreeProof = (leaves: string[], index: number): string[] => {
  const parsedLeaves = []
  for (let i = 0; i < Math.pow(2, Math.ceil(Math.log2(leaves.length))); i++) {
    if (i < leaves.length) {
      parsedLeaves.push(leaves[i])
    } else {
      parsedLeaves.push(ethers.utils.keccak256('0x' + '00'.repeat(32)))
    }
  }

  const bufLeaves = parsedLeaves.map(fromHexString)

  const tree = new MerkleTree(
    bufLeaves,
    (el: Buffer | string): Buffer => {
      return fromHexString(ethers.utils.keccak256(el))
    }
  )

  const proof = tree.getProof(bufLeaves[index], index).map((element: any) => {
    return toHexString(element.data)
  })

  return proof
}

const getStateTrieProof = async (
  l2RpcProvider: ethers.providers.JsonRpcProvider,
  blockNumber: number,
  address: string,
  slot: string
): Promise<StateTrieProof> => {
  const proof = await l2RpcProvider.send('eth_getProof', [
    address,
    [slot],
    toRpcHexString(blockNumber),
  ])

  return {
    accountProof: toHexString(rlp.encode(proof.accountProof)),
    storageProof: toHexString(rlp.encode(proof.storageProof[0].proof)),
  }
}

/**
 * Generates the transaction data to send to the L1CrossDomainMessenger in order to execute an
 * L2 => L1 message.
 * @param l2RpcProviderUrl
 * @param l2CrossDomainMessengerAddress
 * @param l2TransactionHash
 * @param l2BlockOffset
 * @returns 0x-prefixed transaction data as a hex string.
 */
export const makeRelayTransactionData = async (
  l1RpcProviderUrl: string,
  l2RpcProviderUrl: string,
  l1StateCommitmentChainAddress: string,
  l2CrossDomainMessengerAddress: string,
  l2TransactionHash: string
): Promise<string> => {
  const l1RpcProvider = new ethers.providers.JsonRpcProvider(l1RpcProviderUrl)
  const l2RpcProvider = new ethers.providers.JsonRpcProvider(l2RpcProviderUrl)

  const l2Transaction = await l2RpcProvider.getTransaction(l2TransactionHash)
  if (l2Transaction === null) {
    throw new Error(`unable to find tx with hash: ${l2TransactionHash}`)
  }

  const message = await getMessageByTransactionHash(
    l2RpcProvider,
    l2CrossDomainMessengerAddress,
    l2TransactionHash
  )
  if (message === null) {
    throw new Error(
      `unable to find a message to relay in tx with hash: ${l2TransactionHash}`
    )
  }

  const messageSlot = ethers.utils.keccak256(
    ethers.utils.keccak256(
      encodeCrossDomainMessage(message) +
        remove0x(l2CrossDomainMessengerAddress)
    ) + '00'.repeat(32)
  )

  const stateTrieProof = await getStateTrieProof(
    l2RpcProvider,
    l2Transaction.blockNumber + NUM_L2_GENESIS_BLOCKS,
    l2CrossDomainMessengerAddress,
    messageSlot
  )

  const batch = await getStateRootBatchByIndex(
    l1RpcProvider,
    l1StateCommitmentChainAddress,
    l2Transaction.blockNumber - NUM_L2_GENESIS_BLOCKS
  )

  const txIndexInBatch =
    l2Transaction.blockNumber - batch.header.prevTotalElements.toNumber()

  const stateRootMerkleProof = getMerkleTreeProof(
    batch.stateRoots,
    txIndexInBatch
  )

  const relayTransactionData = getContractInterface(
    'OVM_L1CrossDomainMessenger'
  ).encodeFunctionData('relayMessage', [
    message.target,
    message.sender,
    message.message,
    message.messageNonce,
    {
      stateRoot: batch.stateRoots[txIndexInBatch],
      stateRootBatchHeader: batch.header,
      stateRootProof: {
        index: txIndexInBatch,
        siblings: stateRootMerkleProof,
      },
      stateTrieWitness: stateTrieProof.accountProof,
      storageTrieWitness: stateTrieProof.storageProof,
    },
  ])

  return relayTransactionData
}
