/* External Imports */
import { BigNumber, Signer } from 'ethers'
import {
  TransactionResponse,
  TransactionReceipt,
} from '@ethersproject/abstract-provider'
import {
  getContractInterface,
  getContractFactory,
} from '@eth-optimism/contracts'
import { OptimismProvider } from '@eth-optimism/provider'
import { Logger } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  CanonicalTransactionChainContract,
  encodeAppendSequencerBatch,
  BatchContext,
  AppendSequencerBatchParams,
} from '../transaciton-chain-contract'
import {
  EIP155TxData,
  CreateEOATxData,
  TxType,
  ctcCoder,
  EthSignTxData,
  txTypePlainText,
} from '../coders'
import {
  L2Block,
  BatchElement,
  Batch,
  QueueOrigin,
  queueOriginPlainText,
} from '..'
import { RollupInfo, Range, BatchSubmitter, BLOCK_OFFSET } from '.'

export class TransactionBatchSubmitter extends BatchSubmitter {
  protected chainContract: CanonicalTransactionChainContract
  protected l2ChainId: number
  protected syncing: boolean
  protected lastL1BlockNumber: number
  private disableQueueBatchAppend: boolean

  constructor(
    signer: Signer,
    l2Provider: OptimismProvider,
    minTxSize: number,
    maxTxSize: number,
    maxBatchSize: number,
    numConfirmations: number,
    finalityConfirmations: number,
    pullFromAddressManager: boolean,
    log: Logger,
    disableQueueBatchAppend: boolean
  ) {
    super(
      signer,
      l2Provider,
      minTxSize,
      maxTxSize,
      maxBatchSize,
      numConfirmations,
      finalityConfirmations,
      pullFromAddressManager,
      log
    )
    this.disableQueueBatchAppend = disableQueueBatchAppend
  }

  /*****************************
   * Batch Submitter Overrides *
   ****************************/

  public async _updateChainInfo(): Promise<void> {
    const info: RollupInfo = await this._getRollupInfo()
    if (info.mode === 'verifier') {
      this.log.error(
        'Verifier mode enabled! Batch submitter only compatible with sequencer mode'
      )
      process.exit(1)
    }
    this.syncing = info.syncing
    const addrs = await this._getChainAddresses(info)
    const ctcAddress = addrs.ctcAddress

    if (
      typeof this.chainContract !== 'undefined' &&
      ctcAddress === this.chainContract.address
    ) {
      return
    }

    const unwrapped_OVM_CanonicalTransactionChain = (
      await getContractFactory('OVM_CanonicalTransactionChain', this.signer)
    ).attach(ctcAddress)

    this.chainContract = new CanonicalTransactionChainContract(
      unwrapped_OVM_CanonicalTransactionChain.address,
      getContractInterface('OVM_CanonicalTransactionChain'),
      this.signer
    )
    this.log.info(
      `Initialized new CTC with address: ${this.chainContract.address}`
    )
    return
  }

  public async _onSync(): Promise<TransactionReceipt> {
    const pendingQueueElements = await this.chainContract.getNumPendingQueueElements()

    if (pendingQueueElements !== 0) {
      this.log.info(
        `Syncing mode enabled! Skipping batch submission and clearing ${pendingQueueElements} queue elements`
      )

      if (!this.disableQueueBatchAppend) {
        // Empty the queue with a huge `appendQueueBatch(..)` call
        return this._submitAndLogTx(
          this.chainContract.appendQueueBatch(99999999),
          'Cleared queue!'
        )
      }
    }
    this.log.info('Syncing mode enabled but queue is empty. Skipping...')
    return
  }

  // TODO: Remove this function and use geth for lastL1BlockNumber!
  private async _updateLastL1BlockNumber() {
    const pendingQueueElements = await this.chainContract.getNumPendingQueueElements()

    if (pendingQueueElements !== 0) {
      const nextQueueIndex = await this.chainContract.getNextQueueIndex()
      const queueElement = await this.chainContract.getQueueElement(
        nextQueueIndex
      )
      this.lastL1BlockNumber = queueElement[2] // The block number is the 3rd element returned in the array....
    } else {
      const curBlockNum = await this.chainContract.provider.getBlockNumber()
      if (!this.lastL1BlockNumber) {
        // Set the block number to the current l1BlockNumber
        this.lastL1BlockNumber = curBlockNum
      } else {
        if (curBlockNum - this.lastL1BlockNumber > 30) {
          // If the lastL1BlockNumber is too old, then set it to a recent
          // block number. (10 blocks ago to prevent reorgs)
          this.lastL1BlockNumber = curBlockNum - 10
        }
      }
    }
  }

  public async _getBatchStartAndEnd(): Promise<Range> {
    const startBlock =
      (await this.chainContract.getTotalElements()).toNumber() + BLOCK_OFFSET // TODO: Remove BLOCK_OFFSET by adding a tx to Geth's genesis
    const endBlock =
      Math.min(
        startBlock + this.maxBatchSize,
        await this.l2Provider.getBlockNumber()
      ) + 1 // +1 because the `endBlock` is *exclusive*
    if (startBlock >= endBlock) {
      if (startBlock > endBlock) {
        this.log
          .error(`More chain elements in L1 (${startBlock}) than in the L2 node (${endBlock}).
                   This shouldn't happen because we don't submit batches if the sequencer is syncing.`)
      }
      this.log.info(`No txs to submit. Skipping batch submission...`)
      return
    }
    return {
      start: startBlock,
      end: endBlock,
    }
  }

  public async _submitBatch(
    startBlock: number,
    endBlock: number
  ): Promise<TransactionReceipt> {
    const batchParams = await this._generateSequencerBatchParams(
      startBlock,
      endBlock
    )
    this.log.debug('Submitting batch. Tx calldata:', batchParams)
    return this._submitAndLogTx(
      this.chainContract.appendSequencerBatch(batchParams),
      'Submitted batch!'
    )
  }

  /*********************
   * Private Functions *
   ********************/

  private async _generateSequencerBatchParams(
    startBlock: number,
    endBlock: number
  ): Promise<AppendSequencerBatchParams> {
    // Get all L2 BatchElements for the given range
    // For now we need to update our internal `lastL1BlockNumber` value
    // which is used when submitting batches.
    this._updateLastL1BlockNumber() // TODO: Remove this
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
    let lastTimestamp = 0
    let lastBlockNumber = 0
    const groupedBlocks: Array<{
      sequenced: BatchElement[]
      queued: BatchElement[]
    }> = []
    for (const block of blocks) {
      if (
        (lastBlockIsSequencerTx === false && block.isSequencerTx === true) ||
        groupedBlocks.length === 0 ||
        (block.timestamp !== lastTimestamp && block.isSequencerTx === true) ||
        (block.blockNumber !== lastBlockNumber && block.isSequencerTx === true)
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
      lastTimestamp = block.timestamp
      lastBlockNumber = block.blockNumber
    }
    for (const groupedBlock of groupedBlocks) {
      if (
        groupedBlock.sequenced.length === 0 &&
        groupedBlock.queued.length === 0
      ) {
        throw new Error(
          'Attempted to generate batch context with 0 queued and 0 sequenced txs!'
        )
      }
      contexts.push({
        numSequencedTransactions: groupedBlock.sequenced.length,
        numSubsequentQueueTransactions: groupedBlock.queued.length,
        timestamp:
          groupedBlock.sequenced.length > 0
            ? groupedBlock.sequenced[0].timestamp
            : groupedBlock.queued[0].timestamp,
        blockNumber:
          groupedBlock.sequenced.length > 0
            ? groupedBlock.sequenced[0].blockNumber
            : groupedBlock.queued[0].blockNumber,
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
      shouldStartAtBatch: shouldStartAtIndex - BLOCK_OFFSET, // TODO: Remove BLOCK_OFFSET by adding a tx to Geth's genesis
      totalElementsToAppend,
      contexts,
      transactions,
    }
  }

  private async _getL2BatchElement(blockNumber: number): Promise<BatchElement> {
    const block = await this._getBlock(blockNumber)
    const txType = block.transactions[0].txType

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
        blockNumber: block.transactions[0].l1BlockNumber,
      }
    }
  }

  private async _getBlock(blockNumber: number): Promise<L2Block> {
    const block = (await this.l2Provider.getBlockWithTransactions(
      blockNumber
    )) as L2Block
    // Convert the tx type to a number
    block.transactions[0].txType = txTypePlainText[block.transactions[0].txType]
    block.transactions[0].queueOrigin =
      queueOriginPlainText[block.transactions[0].queueOrigin]
    // For now just set the l1BlockNumber based on the current l1 block number
    if (!block.transactions[0].l1BlockNumber) {
      block.transactions[0].l1BlockNumber = this.lastL1BlockNumber
    }
    return block
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
      sequencerTxType: block.transactions[0].txType,
      txData,
      timestamp: block.timestamp,
      blockNumber: block.transactions[0].l1BlockNumber,
    }
  }

  private _getCreateEoaBatchElement(block: L2Block): BatchElement {
    const txData: CreateEOATxData = ctcCoder.createEOATxData.decode(
      block.transactions[0].data
    )
    return {
      stateRoot: block.stateRoot,
      isSequencerTx: true,
      sequencerTxType: block.transactions[0].txType,
      txData,
      timestamp: block.timestamp,
      blockNumber: block.transactions[0].l1BlockNumber,
    }
  }

  private _isSequencerTx(block: L2Block): boolean {
    return block.transactions[0].queueOrigin === QueueOrigin.Sequencer
  }
}
