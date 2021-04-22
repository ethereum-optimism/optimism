import { ethers, Event, Contract, BigNumber, providers } from 'ethers'
import { MerkleTree } from 'merkletreejs'

import {
  StateRootBatchHeader,
  StateRootBatchProof,
  TransactionBatchHeader,
  TransactionBatchProof,
  TransactionChainElement,
  OvmTransaction,
} from '../../types'

import { fromHexString, toHexString } from '@eth-optimism/core-utils'

export class L1ProviderWrapper {
  private eventCache: {
    [topic: string]: {
      startingBlockNumber: number
      events: ethers.Event[]
    }
  } = {}

  constructor(
    public provider: providers.JsonRpcProvider,
    public OVM_StateCommitmentChain: Contract,
    public OVM_CanonicalTransactionChain: Contract,
    public OVM_ExecutionManager: Contract,
    public l1StartOffset: number,
    public l1BlockFinality: number
  ) {}

  public async findAllEvents(
    contract: Contract,
    filter: ethers.EventFilter,
    fromBlock?: number
  ): Promise<ethers.Event[]> {
    const cache = this.eventCache[filter.topics[0] as string] || {
      startingBlockNumber: fromBlock || this.l1StartOffset,
      events: [],
    }

    let events: ethers.Event[] = []
    let startingBlockNumber = cache.startingBlockNumber
    let latestL1BlockNumber = await this.provider.getBlockNumber()
    while (startingBlockNumber < latestL1BlockNumber) {
      events = events.concat(
        await contract.queryFilter(
          filter,
          startingBlockNumber,
          Math.min(
            startingBlockNumber + 2000,
            latestL1BlockNumber - this.l1BlockFinality
          )
        )
      )

      if (startingBlockNumber + 2000 > latestL1BlockNumber) {
        cache.startingBlockNumber = latestL1BlockNumber
        cache.events = cache.events.concat(events)
        break
      }

      startingBlockNumber += 2000
      latestL1BlockNumber = await this.provider.getBlockNumber()
    }

    this.eventCache[filter.topics[0] as string] = cache

    return cache.events
  }

  public async getStateRootBatchHeader(
    index: number
  ): Promise<StateRootBatchHeader> {
    const event = await this._getStateRootBatchEvent(index)

    if (!event) {
      return
    }

    return {
      batchIndex: event.args._batchIndex,
      batchRoot: event.args._batchRoot,
      batchSize: event.args._batchSize,
      prevTotalElements: event.args._prevTotalElements,
      extraData: event.args._extraData,
    }
  }

  public async getStateRoot(index: number): Promise<string> {
    const stateRootBatchHeader = await this.getStateRootBatchHeader(index)
    if (stateRootBatchHeader === undefined) {
      return
    }

    const batchStateRoots = await this.getBatchStateRoots(index)

    return batchStateRoots[
      index - stateRootBatchHeader.prevTotalElements.toNumber()
    ]
  }

  public async getBatchStateRoots(index: number): Promise<string[]> {
    const event = await this._getStateRootBatchEvent(index)

    if (!event) {
      return
    }

    const transaction = await this.provider.getTransaction(
      event.transactionHash
    )

    const [
      stateRoots,
    ] = this.OVM_StateCommitmentChain.interface.decodeFunctionData(
      'appendStateBatch',
      transaction.data
    )

    return stateRoots
  }

  public async getStateRootBatchProof(
    index: number
  ): Promise<StateRootBatchProof> {
    const batchHeader = await this.getStateRootBatchHeader(index)
    const stateRoots = await this.getBatchStateRoots(index)

    const elements = []
    for (
      let i = 0;
      i < Math.pow(2, Math.ceil(Math.log2(stateRoots.length)));
      i++
    ) {
      if (i < stateRoots.length) {
        elements.push(stateRoots[i])
      } else {
        elements.push(ethers.utils.keccak256('0x' + '00'.repeat(32)))
      }
    }

    const hash = (el: Buffer | string): Buffer => {
      return Buffer.from(ethers.utils.keccak256(el).slice(2), 'hex')
    }

    // State roots aren't hashed since they're already bytes32
    const leaves = elements.map((element) => {
      return fromHexString(element)
    })

    const tree = new MerkleTree(leaves, hash)
    const batchIndex = index - batchHeader.prevTotalElements.toNumber()
    const treeProof = tree
      .getProof(leaves[batchIndex], batchIndex)
      .map((element) => {
        return element.data
      })

    return {
      stateRoot: stateRoots[batchIndex],
      stateRootBatchHeader: batchHeader,
      stateRootProof: {
        index: batchIndex,
        siblings: treeProof,
      },
    }
  }

  public async getTransactionBatchHeader(
    index: number
  ): Promise<TransactionBatchHeader> {
    const event = await this._getTransactionBatchEvent(index)

    if (!event) {
      return
    }

    return {
      batchIndex: event.args._batchIndex,
      batchRoot: event.args._batchRoot,
      batchSize: event.args._batchSize,
      prevTotalElements: event.args._prevTotalElements,
      extraData: event.args._extraData,
    }
  }

  public async getBatchTransactions(
    index: number
  ): Promise<
    {
      transaction: OvmTransaction
      transactionChainElement: TransactionChainElement
    }[]
  > {
    const event = await this._getTransactionBatchEvent(index)

    if (!event) {
      return
    }

    const emGasLimit = await this.OVM_ExecutionManager.getMaxTransactionGasLimit()

    const transaction = await this.provider.getTransaction(
      event.transactionHash
    )

    if ((event as any).isSequencerBatch) {
      const transactions = []
      const txdata = fromHexString(transaction.data)
      const shouldStartAtBatch = BigNumber.from(txdata.slice(4, 9))
      const totalElementsToAppend = BigNumber.from(txdata.slice(9, 12))
      const numContexts = BigNumber.from(txdata.slice(12, 15))

      let nextTxPointer = 15 + 16 * numContexts.toNumber()
      for (let i = 0; i < numContexts.toNumber(); i++) {
        const contextPointer = 15 + 16 * i
        const context = {
          numSequencedTransactions: BigNumber.from(
            txdata.slice(contextPointer, contextPointer + 3)
          ),
          numSubsequentQueueTransactions: BigNumber.from(
            txdata.slice(contextPointer + 3, contextPointer + 6)
          ),
          ctxTimestamp: BigNumber.from(
            txdata.slice(contextPointer + 6, contextPointer + 11)
          ),
          ctxBlockNumber: BigNumber.from(
            txdata.slice(contextPointer + 11, contextPointer + 16)
          ),
        }

        for (let j = 0; j < context.numSequencedTransactions.toNumber(); j++) {
          const txDataLength = BigNumber.from(
            txdata.slice(nextTxPointer, nextTxPointer + 3)
          )
          const txData = txdata.slice(
            nextTxPointer + 3,
            nextTxPointer + 3 + txDataLength.toNumber()
          )

          transactions.push({
            transaction: {
              blockNumber: context.ctxBlockNumber.toNumber(),
              timestamp: context.ctxTimestamp.toNumber(),
              gasLimit: emGasLimit,
              entrypoint: '0x4200000000000000000000000000000000000005',
              l1TxOrigin: '0x' + '00'.repeat(20),
              l1QueueOrigin: 0,
              data: toHexString(txData),
            },
            transactionChainElement: {
              isSequenced: true,
              queueIndex: 0,
              timestamp: context.ctxTimestamp.toNumber(),
              blockNumber: context.ctxBlockNumber.toNumber(),
              txData: toHexString(txData),
            },
          })

          nextTxPointer += 3 + txDataLength.toNumber()
        }
      }

      return transactions
    } else {
      return []
    }
  }

  public async getTransactionBatchProof(
    index: number
  ): Promise<TransactionBatchProof> {
    const batchHeader = await this.getTransactionBatchHeader(index)
    const transactions = await this.getBatchTransactions(index)

    const elements = []
    for (
      let i = 0;
      i < Math.pow(2, Math.ceil(Math.log2(transactions.length)));
      i++
    ) {
      if (i < transactions.length) {
        // TODO: FIX
        const tx = transactions[i]
        elements.push(
          `0x01${BigNumber.from(tx.transaction.timestamp)
            .toHexString()
            .slice(2)
            .padStart(64, '0')}${BigNumber.from(tx.transaction.blockNumber)
            .toHexString()
            .slice(2)
            .padStart(64, '0')}${tx.transaction.data.slice(2)}`
        )
      } else {
        elements.push('0x' + '00'.repeat(32))
      }
    }

    const hash = (el: Buffer | string): Buffer => {
      return Buffer.from(ethers.utils.keccak256(el).slice(2), 'hex')
    }

    const leaves = elements.map((element) => {
      return hash(element)
    })

    const tree = new MerkleTree(leaves, hash)
    const batchIndex = index - batchHeader.prevTotalElements.toNumber()
    const treeProof = tree
      .getProof(leaves[batchIndex], batchIndex)
      .map((element) => {
        return element.data
      })

    return {
      transaction: transactions[batchIndex].transaction,
      transactionChainElement: transactions[batchIndex].transactionChainElement,
      transactionBatchHeader: batchHeader,
      transactionProof: {
        index: batchIndex,
        siblings: treeProof,
      },
    }
  }

  private async _getStateRootBatchEvent(index: number): Promise<Event> {
    const events = await this.findAllEvents(
      this.OVM_StateCommitmentChain,
      this.OVM_StateCommitmentChain.filters.StateBatchAppended()
    )

    if (events.length === 0) {
      return
    }

    const matching = events.filter((event) => {
      return (
        event.args._prevTotalElements.toNumber() <= index &&
        event.args._prevTotalElements.toNumber() +
          event.args._batchSize.toNumber() >
          index
      )
    })

    const deletions = await this.findAllEvents(
      this.OVM_StateCommitmentChain,
      this.OVM_StateCommitmentChain.filters.StateBatchDeleted()
    )

    const results: ethers.Event[] = []
    for (const event of matching) {
      const wasDeleted = deletions.some((deletion) => {
        return (
          deletion.blockNumber > event.blockNumber &&
          deletion.args._batchIndex.toNumber() ===
            event.args._batchIndex.toNumber()
        )
      })

      if (!wasDeleted) {
        results.push(event)
      }
    }

    if (results.length === 0) {
      return
    }

    if (results.length > 2) {
      throw new Error(
        `Found more than one batch header for the same state root, this shouldn't happen.`
      )
    }

    return results[results.length - 1]
  }

  private async _getTransactionBatchEvent(
    index: number
  ): Promise<Event & { isSequencerBatch: boolean }> {
    const events = await this.findAllEvents(
      this.OVM_CanonicalTransactionChain,
      this.OVM_CanonicalTransactionChain.filters.TransactionBatchAppended()
    )

    if (events.length === 0) {
      return
    }

    // tslint:disable-next-line
    const event = events.find((event) => {
      return (
        event.args._prevTotalElements.toNumber() <= index &&
        event.args._prevTotalElements.toNumber() +
          event.args._batchSize.toNumber() >
          index
      )
    })

    if (!event) {
      return
    }

    const batchSubmissionEvents = await this.findAllEvents(
      this.OVM_CanonicalTransactionChain,
      this.OVM_CanonicalTransactionChain.filters.SequencerBatchAppended()
    )

    if (batchSubmissionEvents.length === 0) {
      ;(event as any).isSequencerBatch = false
    } else {
      // tslint:disable-next-line
      const batchSubmissionEvent = batchSubmissionEvents.find((event) => {
        return (
          event.args._startingQueueIndex.toNumber() <= index &&
          event.args._startingQueueIndex.toNumber() +
            event.args._totalElements.toNumber() >
            index
        )
      })

      if (batchSubmissionEvent) {
        ;(event as any).isSequencerBatch = true
      } else {
        ;(event as any).isSequencerBatch = false
      }
    }

    return event as any
  }
}
