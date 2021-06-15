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

interface CrossDomainMessageProof {
  stateRoot: string
  stateRootBatchHeader: StateRootBatchHeader
  stateRootProof: {
    index: number
    siblings: string[]
  }
  stateTrieWitness: string
  storageTrieWitness: string
}

interface CrossDomainMessagePair {
  message: CrossDomainMessage
  proof: CrossDomainMessageProof
}

interface StateTrieProof {
  accountProof: string
  storageProof: string
}

/**
 * Finds all L2 => L1 messages triggered by a given L2 transaction, if the message exists.
 *
 * @param l2RpcProvider L2 RPC provider.
 * @param l2CrossDomainMessengerAddress Address of the L2CrossDomainMessenger.
 * @param l2TransactionHash Hash of the L2 transaction to find a message for.
 * @returns Messages associated with the transaction.
 */
export const getMessagesByTransactionHash = async (
  l2RpcProvider: ethers.providers.JsonRpcProvider,
  l2CrossDomainMessengerAddress: string,
  l2TransactionHash: string
): Promise<CrossDomainMessage[]> => {
  // Complain if we can't find the given transaction.
  const transaction = await l2RpcProvider.getTransaction(l2TransactionHash)
  if (transaction === null) {
    throw new Error(`unable to find tx with hash: ${l2TransactionHash}`)
  }

  const l2CrossDomainMessenger = new ethers.Contract(
    l2CrossDomainMessengerAddress,
    getContractInterface('OVM_L2CrossDomainMessenger'),
    l2RpcProvider
  )

  // Find all SentMessage events created in the same block as the given transaction. This is
  // reliable because we should only have one transaction per block.
  const sentMessageEvents = await l2CrossDomainMessenger.queryFilter(
    l2CrossDomainMessenger.filters.SentMessage(),
    transaction.blockNumber,
    transaction.blockNumber
  )

  // Decode the messages and turn them into a nicer struct.
  const sentMessages = sentMessageEvents.map((sentMessageEvent) => {
    const encodedMessage = sentMessageEvent.args.message
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
  })

  return sentMessages
}

/**
 * Encodes a cross domain message.
 *
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
 *
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

  const isEventHi = (event: ethers.Event, index: number) => {
    const prevTotalElements = event.args._prevTotalElements.toNumber()
    return index < prevTotalElements
  }

  const isEventLo = (event: ethers.Event, index: number) => {
    const prevTotalElements = event.args._prevTotalElements.toNumber()
    const batchSize = event.args._batchSize.toNumber()
    return index >= prevTotalElements + batchSize
  }

  const totalBatches: ethers.BigNumber = await l1StateCommitmentChain.getTotalBatches()
  if (totalBatches.eq(0)) {
    return null
  }

  let lowerBound = 0
  let upperBound = totalBatches.toNumber() - 1
  let batchEvent: ethers.Event | null = await getStateBatchAppendedEventByBatchIndex(
    upperBound
  )

  if (isEventLo(batchEvent, l2TransactionIndex)) {
    // Upper bound is too low, means this transaction doesn't have a corresponding state batch yet.
    return null
  } else if (!isEventHi(batchEvent, l2TransactionIndex)) {
    // Upper bound is not too low and also not too high. This means the upper bound event is the
    // one we're looking for! Return it.
    return batchEvent
  }

  // Binary search to find the right event. The above checks will guarantee that the event does
  // exist and that we'll find it during this search.
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
 *
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
 *
 * @param leaves Leaves of the merkle tree.
 * @param index Index to generate a proof for.
 * @returns Merkle proof sibling leaves, as hex strings.
 */
const getMerkleTreeProof = (leaves: string[], index: number): string[] => {
  // Our specific Merkle tree implementation requires that the number of leaves is a power of 2.
  // If the number of given leaves is less than a power of 2, we need to round up to the next
  // available power of 2. We fill the remaining space with the hash of bytes32(0).
  const correctedTreeSize = Math.pow(2, Math.ceil(Math.log2(leaves.length)))
  const parsedLeaves = []
  for (let i = 0; i < correctedTreeSize; i++) {
    if (i < leaves.length) {
      parsedLeaves.push(leaves[i])
    } else {
      parsedLeaves.push(ethers.utils.keccak256('0x' + '00'.repeat(32)))
    }
  }

  // merkletreejs prefers things to be Buffers.
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
 *
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
 * Finds all L2 => L1 messages sent in a given L2 transaction and generates proofs for each of
 * those messages.
 *
 * @param l1RpcProvider L1 RPC provider.
 * @param l2RpcProvider L2 RPC provider.
 * @param l1StateCommitmentChainAddress Address of the StateCommitmentChain.
 * @param l2CrossDomainMessengerAddress Address of the L2CrossDomainMessenger.
 * @param l2TransactionHash L2 transaction hash to generate a relay transaction for.
 * @returns An array of messages sent in the transaction and a proof of inclusion for each.
 */
export const getMessagesAndProofsForL2Transaction = async (
  l1RpcProvider: ethers.providers.JsonRpcProvider | string,
  l2RpcProvider: ethers.providers.JsonRpcProvider | string,
  l1StateCommitmentChainAddress: string,
  l2CrossDomainMessengerAddress: string,
  l2TransactionHash: string
): Promise<CrossDomainMessagePair[]> => {
  if (typeof l1RpcProvider === 'string') {
    l1RpcProvider = new ethers.providers.JsonRpcProvider(l1RpcProvider)
  }
  if (typeof l2RpcProvider === 'string') {
    l2RpcProvider = new ethers.providers.JsonRpcProvider(l2RpcProvider)
  }

  const l2Transaction = await l2RpcProvider.getTransaction(l2TransactionHash)
  if (l2Transaction === null) {
    throw new Error(`unable to find tx with hash: ${l2TransactionHash}`)
  }

  // Need to find the state batch for the given transaction. If no state batch has been published
  // yet then we will not be able to generate a proof.
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

  // Adjust the transaction index based on the number of L2 genesis block we have. "Index" here
  // refers to the position of the transaction within the *Canonical Transaction Chain*.
  const l2TransactionIndex = l2Transaction.blockNumber - NUM_L2_GENESIS_BLOCKS

  // Here the index refers to the position of the state root that corresponds to this transaction
  // within the batch of state roots in which that state root was published.
  const txIndexInBatch =
    l2TransactionIndex - batch.header.prevTotalElements.toNumber()

  // Find every message that was sent during this transaction. We'll then attach a proof for each.
  const messages = await getMessagesByTransactionHash(
    l2RpcProvider,
    l2CrossDomainMessengerAddress,
    l2TransactionHash
  )

  const messagePairs: CrossDomainMessagePair[] = []
  for (const message of messages) {
    // We need to calculate the specific storage slot that demonstrates that this message was
    // actually included in the L2 chain. The following calculation is based on the fact that
    // messages are stored in the following mapping on L2:
    // https://github.com/ethereum-optimism/optimism/blob/c84d3450225306abbb39b4e7d6d82424341df2be/packages/contracts/contracts/optimistic-ethereum/OVM/predeploys/OVM_L2ToL1MessagePasser.sol#L23
    // You can read more about how Solidity storage slots are computed for mappings here:
    // https://docs.soliditylang.org/en/v0.8.4/internals/layout_in_storage.html#mappings-and-dynamic-arrays
    const messageSlot = ethers.utils.keccak256(
      ethers.utils.keccak256(
        encodeCrossDomainMessage(message) +
          remove0x(l2CrossDomainMessengerAddress)
      ) + '00'.repeat(32)
    )

    // We need a Merkle trie proof for the given storage slot. This allows us to prove to L1 that
    // the message was actually sent on L2.
    const stateTrieProof = await getStateTrieProof(
      l2RpcProvider,
      l2Transaction.blockNumber,
      predeploys.OVM_L2ToL1MessagePasser,
      messageSlot
    )

    // State roots are published in batches to L1 and correspond 1:1 to transactions. We compute a
    // Merkle root for these state roots so that we only need to store the minimum amount of
    // information on-chain. So we need to create a Merkle proof for the specific state root that
    // corresponds to this transaction.
    const stateRootMerkleProof = getMerkleTreeProof(
      batch.stateRoots,
      txIndexInBatch
    )

    // We now have enough information to create the message proof.
    const proof: CrossDomainMessageProof = {
      stateRoot: batch.stateRoots[txIndexInBatch],
      stateRootBatchHeader: batch.header,
      stateRootProof: {
        index: txIndexInBatch,
        siblings: stateRootMerkleProof,
      },
      stateTrieWitness: stateTrieProof.accountProof,
      storageTrieWitness: stateTrieProof.storageProof,
    }

    messagePairs.push({
      message,
      proof,
    })
  }

  return messagePairs
}
