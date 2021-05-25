/* Imports: External */
import { ethers } from 'ethers'
import {
  fromHexString,
  remove0x,
  toHexString,
  toRpcHexString,
} from '@eth-optimism/core-utils'
import { getContractInterface, predeploys } from '@eth-optimism/contracts'
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

/**
 * Finds the L2 => L1 message triggered by a given L2 transaction, if the message exists.
 * @param l2RpcProvider L2 RPC provider.
 * @param l2CrossDomainMessengerAddress Address of the L2CrossDomainMessenger.
 * @param l2TransactionHash Hash of the L2 transaction to find a message for.
 * @returns Message assocaited with the transaction or null if no such message exists.
 */
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

/**
 * Encodes a cross domain message.
 * @param message Message to encode.
 * @returns Encoded message.
 */
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

/**
 * Finds the StateBatchAppended event associated with a given L2 transaction.
 * @param l1RpcProvider L1 RPC provider.
 * @param l1StateCommitmentChainAddress Address of the L1StateCommitmentChain.
 * @param l2TransactionIndex Index of the L2 transaction to find a StateBatchAppended event for.
 * @returns StateBatchAppended event for the given transaction or null if no such event exists.
 */
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

/**
 * Finds the full state root batch associated with a given transaction index.
 * @param l1RpcProvider L1 RPC provider.
 * @param l1StateCommitmentChainAddress Address of the L1StateCommitmentChain.
 * @param l2TransactionIndex Index of the L2 transaction to find a state root batch for.
 * @returns State root batch associated with the given transaction index or null if no state root
 * batch exists.
 */
export const getStateRootBatchByTransactionIndex = async (
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

/**
 * Generates a Merkle proof (using the particular scheme we use within Lib_MerkleTree).
 * @param leaves Leaves of the merkle tree.
 * @param index Index to generate a proof for.
 * @returns Merkle proof sibling leaves, as hex strings.
 */
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

/**
 * Generates a Merkle-Patricia trie proof for a given account and storage slot.
 * @param l2RpcProvider L2 RPC provider.
 * @param blockNumber Block number to generate the proof at.
 * @param address Address to generate the proof for.
 * @param slot Storage slot to generate the proof for.
 * @returns Account proof and storage proof.
 */
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
 * @param l1RpcProviderUrl L1 RPC provider url.
 * @param l2RpcProviderUrl L2 RPC provider url.
 * @param l1StateCommitmentChainAddress Address of the StateCommitmentChain.
 * @param l2CrossDomainMessengerAddress Address of the L2CrossDomainMessenger.
 * @param l2TransactionHash L2 transaction hash to generate a relay transaction for.
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

  // Step 1: Find the transaction.
  const l2Transaction = await l2RpcProvider.getTransaction(l2TransactionHash)
  if (l2Transaction === null) {
    throw new Error(`unable to find tx with hash: ${l2TransactionHash}`)
  }

  // Step 2: Find the message associated with the transaction.
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

  // Step 3: Generate a state trie proof for the slot where the message is located.
  const messageSlot = ethers.utils.keccak256(
    ethers.utils.keccak256(
      encodeCrossDomainMessage(message) +
        remove0x(l2CrossDomainMessengerAddress)
    ) + '00'.repeat(32)
  )
  const stateTrieProof = await getStateTrieProof(
    l2RpcProvider,
    l2Transaction.blockNumber + NUM_L2_GENESIS_BLOCKS,
    predeploys.OVM_L2ToL1MessagePasser,
    messageSlot
  )

  // Step 4: Find the full batch associated with the transaction.
  const batch = await getStateRootBatchByTransactionIndex(
    l1RpcProvider,
    l1StateCommitmentChainAddress,
    l2Transaction.blockNumber - NUM_L2_GENESIS_BLOCKS
  )
  if (batch === null) {
    throw new Error(
      `unable to find state root batch for tx with hash: ${l2TransactionHash}`
    )
  }

  // Step 5: Generate a Merkle proof for the state root associated with the transaction inside of
  // the Merkle tree of state roots published as a batch.
  const txIndexInBatch =
    l2Transaction.blockNumber - batch.header.prevTotalElements.toNumber()
  const stateRootMerkleProof = getMerkleTreeProof(
    batch.stateRoots,
    txIndexInBatch
  )

  // Step 6: Generate the transaction data.
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
