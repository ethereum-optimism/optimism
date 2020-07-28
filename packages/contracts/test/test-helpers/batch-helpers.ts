import { Contract, Signer } from "ethers"
import { TxChainBatch, TxQueueBatch, StateChainBatch } from "./types"

export function makeRepeatedBytes(value: string, length: number): string {
  const repeated = value.repeat((length * 2) / value.length + 1)
  return '0x' + repeated.slice(0, length * 2)
}

export function makeRandomBlockOfSize(blockSize: number): string[] {
  const block = []
  for (let i = 0; i < blockSize; i++) {
    block.push(makeRepeatedBytes('' + Math.floor(Math.random() * 500 + 1), 32))
  }
  return block
}

export function makeRandomBatchOfSize(batchSize: number): string[] {
  return makeRandomBlockOfSize(batchSize)
}

export const appendSequencerBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[]
): Promise<number> => {
  const timestamp = Math.floor(Date.now() / 1000)
  // Submit the rollup batch on-chain
  await canonicalTransactionChain
    .connect(sequencer)
    .appendSequencerBatch(batch, timestamp)
  return timestamp
}

export const appendAndGenerateSequencerBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[],
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<TxChainBatch> => {
  const timestamp = await appendSequencerBatch(
    canonicalTransactionChain,
    sequencer,
    batch
  )

  return createTxChainBatch(
    batch,
    timestamp,
    false,
    batchIndex,
    cumulativePrevElements
  )
}

export const createTxChainBatch = async (
  batch: string[],
  timestamp: number,
  isL1ToL2Tx: boolean,
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<TxChainBatch> => {
  const localBatch = new TxChainBatch(
    timestamp,
    isL1ToL2Tx,
    batchIndex,
    cumulativePrevElements,
    batch
  )
  await localBatch.generateTree()
  return localBatch
}

export const enqueueAndGenerateL1ToL2Batch = async (
  provider: any,
  l1ToL2Queue: Contract,
  l1ToL2TransactionPasser: Signer,
  tx: string
): Promise<TxQueueBatch> => {
  // Submit the rollup batch on-chain
  const enqueueTx = await l1ToL2Queue
    .connect(l1ToL2TransactionPasser)
    .enqueueTx(tx)
  const localBatch = await generateQueueBatch(provider, tx, enqueueTx.hash)
  return localBatch
}

export const enqueueAndGenerateSafetyBatch = async (
  provider: any,
  safetyQueue: Contract,
  randomWallet: Signer,
  tx: string
): Promise<TxQueueBatch> => {
  const enqueueTx = await safetyQueue.connect(randomWallet).enqueueTx(tx)
  const localBatch = await generateQueueBatch(provider, tx, enqueueTx.hash)
  return localBatch
}

export const generateQueueBatch = async (
  provider: any,
  tx: string,
  txHash: string
): Promise<TxQueueBatch> => {
  const txReceipt = await provider.getTransactionReceipt(txHash)
  const timestamp = (await provider.getBlock(txReceipt.blockNumber)).timestamp
  // Generate a local version of the rollup batch
  const localBatch = new TxQueueBatch(tx, timestamp)
  await localBatch.generateTree()
  return localBatch
}

export const generateStateBatch = async (
  batch: string[],
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<StateChainBatch> => {
  const localBatch = new StateChainBatch(
    batchIndex,
    cumulativePrevElements,
    batch
  )
  await localBatch.generateTree()
  return localBatch
}

export const generateTxBatch = async (
  batch: string[],
  timestamp: number,
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<TxChainBatch> => {
  const localBatch = new TxChainBatch(
    timestamp,
    false,
    batchIndex,
    cumulativePrevElements,
    batch
  )
  await localBatch.generateTree()
  return localBatch
}

export const appendAndGenerateStateBatch = async (
  stateChain: Contract,
  batch: string[],
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<StateChainBatch> => {
  await stateChain.appendStateBatch(batch)
  // Generate a local version of the rollup batch
  const localBatch = new StateChainBatch(
    batchIndex,
    cumulativePrevElements,
    batch
  )
  await localBatch.generateTree()
  return localBatch
}

export const appendTxBatch = async (
  canonicalTxChain: Contract,
  sequencer: Signer,
  batch: string[]
): Promise<number> => {
  const timestamp = Math.floor(Date.now() / 1000)
  // Submit the rollup batch on-chain
  await canonicalTxChain
    .connect(sequencer)
    .appendSequencerBatch(batch, timestamp)

  return timestamp
}

export const appendAndGenerateTransactionBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[],
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<TxChainBatch> => {
  const timestamp = await appendTxBatch(
    canonicalTransactionChain,
    sequencer,
    batch
  )

  const localBatch = new TxChainBatch(
    timestamp,
    false,
    batchIndex,
    cumulativePrevElements,
    batch
  )

  await localBatch.generateTree()

  return localBatch
}