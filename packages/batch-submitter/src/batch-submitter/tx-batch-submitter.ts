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
} from '../coders'
import { L2Block, BatchElement, Batch, QueueOrigin } from '..'
import { RollupInfo, BatchSubmitter } from '.'

export class TransactionBatchSubmitter extends BatchSubmitter {
  protected chainContract: CanonicalTransactionChainContract
  protected l2ChainId: number
  protected syncing: boolean

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
    const ctcAddress = info.addresses.canonicalTransactionChain

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
    this.log.info(
      'Syncing mode enabled! Skipping batch submission and clearing queue...'
    )
    // Empty the queue with a huge `appendQueueBatch(..)` call
    return this._submitAndLogTx(
      this.chainContract.appendQueueBatch(99999999),
      'Cleared queue!'
    )
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
}
