/* External Imports */
import { BigNumber, Signer } from 'ethers'
import { TransactionResponse } from '@ethersproject/abstract-provider'
import { getLogger } from '@eth-optimism/core-utils'
import { OptimismProvider } from '@eth-optimism/provider'

const log = getLogger('oe:batch-submitter:core')

/* Internal Imports */
import {
  CanonicalTransactionChainContract,
  encodeAppendSequencerBatch,
  BatchContext,
  AppendSequencerBatchParams,
} from './transaciton-chain-contract'
import { EIP155TxData, CreateEOATxData, TxType, ctcCoder } from './coders'
import { L2Block, BatchElement, Batch, QueueOrigin } from '.'
import { remove0x } from './utils'

export class BatchSubmitter {
  public blockCache: {
    [blockNumber: number]: BatchElement
  } = {}

  constructor(
    readonly txChain: CanonicalTransactionChainContract,
    readonly signer: Signer,
    readonly l2Provider: OptimismProvider,
    readonly l2ChainId: number,
    readonly maxTxSize: number,
    readonly numConfirmations: number,
  ) {}

  public async submitNextBatch(): Promise<void> {
    const startBlock = parseInt(await this.txChain.getTotalElements(), 16) + 1
    const endBlock = Math.min(
      startBlock + 100,
      await this.l2Provider.getBlockNumber()
    )
    log.info(
      `Attempting to submit next batch. Start l2 tx index: ${startBlock} - end index: ${endBlock}`
    )
    if (startBlock === endBlock) {
      log.info(`No txs to submit. Skipping...`)
      return
    }

    const batchParams = await this._generateSequencerBatchParams(
      startBlock,
      endBlock
    )
    const txRes = await this.txChain.appendSequencerBatch(batchParams)
    const receipt = await txRes.wait(this.numConfirmations)
    log.info('Submitted batch!')
    log.debug('Tx receipt:', receipt)
  }

  public async _generateSequencerBatchParams(
    startBlock: number,
    endBlock: number
  ): Promise<AppendSequencerBatchParams> {
    // Get all L2Blocks between the given range
    const blocks: Batch = []
    for (let i = startBlock; i < endBlock; i++) {
      if (!this.blockCache.hasOwnProperty(i)) {
        this.blockCache[i] = await this._getL2Block(i)
      }
      blocks.push(this.blockCache[i])
    }
    let sequencerBatchParams = await this._getSequencerBatchParams(
      startBlock,
      blocks
    )
    let encoded = encodeAppendSequencerBatch(sequencerBatchParams)
    while (encoded.length / 2 > this.maxTxSize) {
      blocks.splice(Math.ceil((blocks.length * 2) / 3)) // Delete 1/3rd of all of the blocks
      sequencerBatchParams = await this._getSequencerBatchParams(
        startBlock,
        blocks
      )
      encoded = encodeAppendSequencerBatch(sequencerBatchParams)
    }
    return sequencerBatchParams
  }

  public async _getSequencerBatchParams(
    shouldStartAtIndex: number,
    blocks: Batch
  ): Promise<AppendSequencerBatchParams> {
    const totalElementsToAppend = blocks.length

    // Generate contexts
    const contexts: BatchContext[] = []
    let lastBlockIsSequencerTx = false
    const groupedBlocks: Array<{
      sequenced: BatchElement[]
      queued: BatchElement[]
    }> = []
    for (const block of blocks) {
      if (
        (lastBlockIsSequencerTx === false && block.isSequencerTx === true) ||
        groupedBlocks.length === 0
      ) {
        groupedBlocks.push({
          sequenced: [],
          queued: [],
        })
      }
      const cur = groupedBlocks.length - 1
      block.isSequencerTx
        ? groupedBlocks[cur].sequenced.push(block)
        : groupedBlocks[cur].queued.push(block)
      lastBlockIsSequencerTx = block.isSequencerTx
    }
    for (const groupedBlock of groupedBlocks) {
      contexts.push({
        numSequencedTransactions: groupedBlock.sequenced.length,
        numSubsequentQueueTransactions: groupedBlock.queued.length,
        timestamp:
          groupedBlock.sequenced.length > 0
            ? groupedBlock.sequenced[0].timestamp
            : 0,
        blockNumber:
          groupedBlock.sequenced.length > 0
            ? groupedBlock.sequenced[0].blockNumber
            : 0,
      })
    }

    // Generate sequencer transactions
    const transactions: string[] = []
    for (const block of blocks) {
      if (!block.isSequencerTx) {
        continue
      }
      let encoding: string = ctcCoder.eip155TxData.encode(
        block.txData as EIP155TxData
      )
      if (block.sequencerTxType === TxType.EIP155) {
        encoding = ctcCoder.eip155TxData.encode(block.txData as EIP155TxData)
      } else if (block.sequencerTxType === TxType.createEOA) {
        encoding = ctcCoder.createEOATxData.encode(
          block.txData as CreateEOATxData
        )
      }
      transactions.push(encoding)
    }

    return {
      shouldStartAtBatch: shouldStartAtIndex - 1,
      totalElementsToAppend,
      contexts,
      transactions,
    }
  }

  public async _getL2Block(blockNumber: number): Promise<BatchElement> {
    const block = (await this.l2Provider.getBlockWithTransactions(
      blockNumber
    )) as L2Block
    const txType = block.transactions[0].meta.txType

    if (this._isSequencerTx(block)) {
      if (txType === TxType.EIP155) {
        return this._getEIP155L2Block(block)
      } else if (txType === TxType.createEOA) {
        return this._getCreateEOAL2Block(block)
      } else {
        throw new Error('Unsupported Tx Type!')
      }
    } else {
      return {
        stateRoot: block.stateRoot,
        isSequencerTx: false,
        sequencerTxType: undefined,
        txData: undefined,
        timestamp: block.timestamp,
        blockNumber: block.number,
      }
    }
  }

  private _getEIP155L2Block(block: L2Block): BatchElement {
    const tx: TransactionResponse = block.transactions[0]
    const txData: EIP155TxData = {
      sig: {
        v: '0' + (tx.v - this.l2ChainId * 2 - 8 - 27).toString(),
        r: tx.r,
        s: tx.s,
      },
      gasLimit: BigNumber.from(tx.gasLimit).toNumber(),
      gasPrice: BigNumber.from(tx.gasPrice).toNumber(),
      nonce: tx.nonce,
      target: tx.to ? tx.to : '00'.repeat(20),
      data: tx.data,
    }
    return {
      stateRoot: block.stateRoot,
      isSequencerTx: true,
      sequencerTxType: block.transactions[0].meta.txType,
      txData,
      timestamp: block.timestamp,
      blockNumber: block.number,
    }
  }

  private _getCreateEOAL2Block(block: L2Block): BatchElement {
    const tx: TransactionResponse = block.transactions[0]
    // Call decode on the data field to get sig and messageHash
    const txData: CreateEOATxData = {
      sig: {
        // TODO: Update v value to strip the chainID
        v: remove0x(BigNumber.from(tx.v).toHexString()).padStart(2, '0'),
        r: tx.r,
        s: tx.s,
      },
      messageHash: tx.data, // TODO: Parse this more
    }
    return {
      stateRoot: block.stateRoot,
      isSequencerTx: true,
      sequencerTxType: block.transactions[0].meta.txType,
      txData,
      timestamp: block.timestamp,
      blockNumber: block.number,
    }
  }

  public _isSequencerTx(block: L2Block): boolean {
    return block.transactions[0].meta.queueOrigin === QueueOrigin.Sequencer
  }
}
