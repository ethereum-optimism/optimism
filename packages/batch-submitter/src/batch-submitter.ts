/* External Imports */
import { BigNumber, Signer } from 'ethers'
import {
  TransactionResponse,
  TransactionReceipt,
} from '@ethersproject/abstract-provider'
import { getLogger } from '@eth-optimism/core-utils'
import { OptimismProvider } from '@eth-optimism/provider'
import {
  getContractInterface,
  getContractFactory,
} from '@eth-optimism/contracts'

const log = getLogger('oe:batch-submitter:core')

/* Internal Imports */
import {
  CanonicalTransactionChainContract,
  encodeAppendSequencerBatch,
  BatchContext,
  AppendSequencerBatchParams,
} from './transaciton-chain-contract'
import {
  EIP155TxData,
  CreateEOATxData,
  TxType,
  ctcCoder,
  EthSignTxData,
  Address,
  Bytes32,
} from './coders'
import { L2Block, BatchElement, Batch, QueueOrigin } from '.'

export interface RollupInfo {
  signer: Address
  mode: 'sequencer' | 'verifier'
  syncing: boolean
  l1BlockHash: Bytes32
  l1BlockHeight: number
  addresses: {
    canonicalTransactionChain: Address
    addressResolver: Address
    l1ToL2TransactionQueue: Address
    sequencerDecompression: Address
  }
}

export class BatchSubmitter {
  private txChain: CanonicalTransactionChainContract
  private l2ChainId: number
  private syncing: boolean

  constructor(
    readonly signer: Signer,
    readonly l2Provider: OptimismProvider,
    readonly maxTxSize: number,
    readonly maxBatchSize: number,
    readonly numConfirmations: number
  ) {}

  public async submitNextBatch(): Promise<TransactionReceipt> {
    await this._updateL2ChainInfo()

    if (this.syncing === true) {
      log.info(
        'Syncing mode enabled! Skipping batch submission and clearing queue...'
      )
      return this._clearQueue()
    }

    const startBlock = parseInt(await this.txChain.getTotalElements(), 16) + 1 // +1 to skip L2 genesis block
    const endBlock = Math.min(
      startBlock + this.maxBatchSize,
      await this.l2Provider.getBlockNumber()
    )
    log.info(
      `Attempting to submit next batch. Start l2 tx index: ${startBlock} - end index: ${endBlock}`
    )
    if (startBlock >= endBlock) {
      if (startBlock > endBlock) {
        log.error(`More txs in CTC (${startBlock}) than in the L2 node (${endBlock}).
                   This shouldn't happen because we don't submit batches if the sequencer is syncing.`)
      }
      log.info(`No txs to submit. Skipping batch submission...`)
      return
    }

    const batchParams = await this._generateSequencerBatchParams(
      startBlock,
      endBlock
    )
    return this._submitAndLogTx(
      this.txChain.appendSequencerBatch(batchParams),
      'Submitted batch!'
    )
  }

  private async _clearQueue(): Promise<TransactionReceipt> {
    // Empty the queue with a huge `appendQueueBatch(..)` call
    return this._submitAndLogTx(
      this.txChain.appendQueueBatch(99999999),
      'Cleared queue!'
    )
  }

  private async _updateL2ChainInfo(): Promise<void> {
    if (typeof this.l2ChainId === 'undefined') {
      this.l2ChainId = await this._getL2ChainId()
    }

    const info: RollupInfo = await this._getRollupInfo()
    if (info.mode === 'verifier') {
      throw new Error(
        'Verifier mode enabled! Batch submitter only compatible with sequencer mode'
      )
    }
    this.syncing = info.syncing
    const ctcAddress = info.addresses.canonicalTransactionChain

    if (
      typeof this.txChain !== 'undefined' &&
      ctcAddress === this.txChain.address
    ) {
      return
    }

    const unwrapped_OVM_CanonicalTransactionChain = (
      await getContractFactory('OVM_CanonicalTransactionChain', this.signer)
    ).attach(ctcAddress)

    this.txChain = new CanonicalTransactionChainContract(
      unwrapped_OVM_CanonicalTransactionChain.address,
      getContractInterface('OVM_CanonicalTransactionChain'),
      this.signer
    )
    log.info(`Initialized new CTC with address: ${this.txChain.address}`)
  }

  private async _generateSequencerBatchParams(
    startBlock: number,
    endBlock: number
  ): Promise<AppendSequencerBatchParams> {
    // Get all L2 BatchElements for the given range
    const batch: Batch = []
    for (let i = startBlock; i < endBlock; i++) {
      batch.push(await this._getL2BatchElement(i))
    }
    let sequencerBatchParams = await this._getSequencerBatchParams(
      startBlock,
      batch
    )
    let encoded = encodeAppendSequencerBatch(sequencerBatchParams)
    while (encoded.length / 2 > this.maxTxSize) {
      batch.splice(Math.ceil((batch.length * 2) / 3)) // Delete 1/3rd of all of the batch elements
      sequencerBatchParams = await this._getSequencerBatchParams(
        startBlock,
        batch
      )
      encoded = encodeAppendSequencerBatch(sequencerBatchParams)
    }
    return sequencerBatchParams
  }

  private async _getSequencerBatchParams(
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
      let encoding: string
      if (block.sequencerTxType === TxType.EIP155) {
        encoding = ctcCoder.eip155TxData.encode(block.txData as EIP155TxData)
      } else if (block.sequencerTxType === TxType.EthSign) {
        encoding = ctcCoder.ethSignTxData.encode(block.txData as EthSignTxData)
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

  private async _getL2BatchElement(blockNumber: number): Promise<BatchElement> {
    const block = (await this.l2Provider.getBlockWithTransactions(
      blockNumber
    )) as L2Block
    const txType = block.transactions[0].meta.txType

    if (this._isSequencerTx(block)) {
      if (txType === TxType.EIP155 || txType === TxType.EthSign) {
        return this._getDefaultEcdsaTxBatchElement(block)
      } else if (txType === TxType.createEOA) {
        return this._getCreateEoaBatchElement(block)
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
        blockNumber: block.transactions[0].meta.l1BlockNumber,
      }
    }
  }

  private _getDefaultEcdsaTxBatchElement(block: L2Block): BatchElement {
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
      blockNumber: block.transactions[0].meta.l1BlockNumber,
    }
  }

  private _getCreateEoaBatchElement(block: L2Block): BatchElement {
    const txData: CreateEOATxData = ctcCoder.createEOATxData.decode(
      block.transactions[0].data
    )
    return {
      stateRoot: block.stateRoot,
      isSequencerTx: true,
      sequencerTxType: block.transactions[0].meta.txType,
      txData,
      timestamp: block.timestamp,
      blockNumber: block.transactions[0].meta.l1BlockNumber,
    }
  }

  private _isSequencerTx(block: L2Block): boolean {
    return block.transactions[0].meta.queueOrigin === QueueOrigin.Sequencer
  }

  private async _getRollupInfo(): Promise<RollupInfo> {
    return this.l2Provider.send('rollup_getInfo', [])
  }

  private async _getL2ChainId(): Promise<number> {
    return this.l2Provider.send('eth_chainId', [])
  }

  private async _submitAndLogTx(
    txPromise: Promise<TransactionResponse>,
    successMessage: string
  ): Promise<TransactionReceipt> {
    const response = await txPromise
    const receipt = await response.wait(this.numConfirmations)
    log.info(successMessage)
    log.debug('Transaction Response:', response)
    log.debug('Transaction receipt:', receipt)
    return receipt
  }
}
