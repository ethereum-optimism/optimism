/* External Imports */
import { Promise as bPromise } from 'bluebird'
import { Contract, Signer, providers } from 'ethers'
import { TransactionReceipt } from '@ethersproject/abstract-provider'
import { getContractFactory } from 'old-contracts'
import {
  L2Block,
  RollupInfo,
  Bytes32,
  remove0x,
} from '@eth-optimism/core-utils'
import { Logger, Metrics } from '@eth-optimism/common-ts'

/* Internal Imports */
import { Range, BatchSubmitter } from '.'

export class StateBatchSubmitter extends BatchSubmitter {
  // TODO: Change this so that we calculate start = scc.totalElements() and end = ctc.totalElements()!
  // Not based on the length of the L2 chain -- that is only used in the batch submitter
  // Note this means we've got to change the state / end calc logic

  protected l2ChainId: number
  protected syncing: boolean
  protected ctcContract: Contract
  private fraudSubmissionAddress: string

  constructor(
    signer: Signer,
    l2Provider: providers.JsonRpcProvider,
    minTxSize: number,
    maxTxSize: number,
    maxBatchSize: number,
    maxBatchSubmissionTime: number,
    numConfirmations: number,
    resubmissionTimeout: number,
    finalityConfirmations: number,
    addressManagerAddress: string,
    minBalanceEther: number,
    minGasPriceInGwei: number,
    maxGasPriceInGwei: number,
    gasRetryIncrement: number,
    gasThresholdInGwei: number,
    blockOffset: number,
    logger: Logger,
    metrics: Metrics,
    fraudSubmissionAddress: string
  ) {
    super(
      signer,
      l2Provider,
      minTxSize,
      maxTxSize,
      maxBatchSize,
      maxBatchSubmissionTime,
      numConfirmations,
      resubmissionTimeout,
      finalityConfirmations,
      addressManagerAddress,
      minBalanceEther,
      minGasPriceInGwei,
      maxGasPriceInGwei,
      gasRetryIncrement,
      gasThresholdInGwei,
      blockOffset,
      logger,
      metrics
    )
    this.fraudSubmissionAddress = fraudSubmissionAddress
  }

  /*****************************
   * Batch Submitter Overrides *
   ****************************/

  public async _updateChainInfo(): Promise<void> {
    const info: RollupInfo = await this._getRollupInfo()
    if (info.mode === 'verifier') {
      this.logger.error(
        'Verifier mode enabled! Batch submitter only compatible with sequencer mode'
      )
      process.exit(1)
    }
    this.syncing = info.syncing
    const addrs = await this._getChainAddresses()
    const sccAddress = addrs.sccAddress
    const ctcAddress = addrs.ctcAddress

    if (
      typeof this.chainContract !== 'undefined' &&
      sccAddress === this.chainContract.address &&
      ctcAddress === this.ctcContract.address
    ) {
      this.logger.debug('Chain contract already initialized', {
        sccAddress,
        ctcAddress,
      })
      return
    }

    this.chainContract = (
      await getContractFactory('OVM_StateCommitmentChain', this.signer)
    ).attach(sccAddress)
    this.ctcContract = (
      await getContractFactory('OVM_CanonicalTransactionChain', this.signer)
    ).attach(ctcAddress)

    this.logger.info('Connected Optimism contracts', {
      stateCommitmentChain: this.chainContract.address,
      canonicalTransactionChain: this.ctcContract.address,
    })
    return
  }

  public async _onSync(): Promise<TransactionReceipt> {
    this.logger.info('Syncing mode enabled! Skipping state batch submission...')
    return
  }

  public async _getBatchStartAndEnd(): Promise<Range> {
    this.logger.info('Getting batch start and end for state batch submitter...')
    const startBlock: number =
      (await this.chainContract.getTotalElements()).toNumber() +
      this.blockOffset
    this.logger.info('Retrieved start block number from SCC', {
      startBlock,
    })

    // We will submit state roots for txs which have been in the tx chain for a while.
    const totalElements: number =
      (await this.ctcContract.getTotalElements()).toNumber() + this.blockOffset
    this.logger.info('Retrieved total elements from CTC', {
      totalElements,
    })

    const endBlock: number = Math.min(
      startBlock + this.maxBatchSize,
      totalElements
    )

    if (startBlock >= endBlock) {
      if (startBlock > endBlock) {
        this.logger.error(
          'State commitment chain is larger than transaction chain. This should never happen!'
        )
      }
      this.logger.info(
        'No state commitments to submit. Skipping batch submission...'
      )
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
    const batch = await this._generateStateCommitmentBatch(startBlock, endBlock)
    const tx = this.chainContract.interface.encodeFunctionData(
      'appendStateBatch',
      [batch, startBlock]
    )
    const batchSizeInBytes = remove0x(tx).length / 2
    this.logger.debug('State batch generated', {
      batchSizeInBytes,
      tx,
    })

    if (!this._shouldSubmitBatch(batchSizeInBytes)) {
      return
    }

    const offsetStartsAtIndex = startBlock - this.blockOffset
    this.logger.debug('Submitting batch.', { tx })

    const nonce = await this.signer.getTransactionCount()
    const contractFunction = async (gasPrice): Promise<TransactionReceipt> => {
      this.logger.info('Submitting appendStateBatch transaction', {
        gasPrice,
        nonce,
        contractAddr: this.chainContract.address,
      })
      const contractTx = await this.chainContract.appendStateBatch(
        batch,
        offsetStartsAtIndex,
        { nonce, gasPrice }
      )
      this.logger.info('Submitted appendStateBatch transaction', {
        txHash: contractTx.hash,
        from: contractTx.from,
      })
      this.logger.debug('appendStateBatch transaction data', {
        data: contractTx.data,
      })
      return this.signer.provider.waitForTransaction(
        contractTx.hash,
        this.numConfirmations
      )
    }
    return this._submitAndLogTx(contractFunction, 'Submitted state root batch!')
  }

  /*********************
   * Private Functions *
   ********************/

  private async _generateStateCommitmentBatch(
    startBlock: number,
    endBlock: number
  ): Promise<Bytes32[]> {
    const blockRange = endBlock - startBlock
    const batch: Bytes32[] = await bPromise.map(
      [...Array(blockRange).keys()],
      async (i: number) => {
        this.logger.debug('Fetching L2BatchElement', {
          blockNo: startBlock + i,
        })
        const block = (await this.l2Provider.getBlockWithTransactions(
          startBlock + i
        )) as L2Block
        const blockTx = block.transactions[0]
        if (blockTx.from === this.fraudSubmissionAddress) {
          this.logger.warn('Found transaction from fraud submission address', {
            txHash: blockTx.hash,
            fraudSubmissionAddress: this.fraudSubmissionAddress,
          })
          this.fraudSubmissionAddress = 'no fraud'
          return '0xbad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1'
        }
        return block.stateRoot
      },
      { concurrency: 100 }
    )

    let tx = this.chainContract.interface.encodeFunctionData(
      'appendStateBatch',
      [batch, startBlock]
    )
    while (remove0x(tx).length / 2 > this.maxTxSize) {
      batch.splice(Math.ceil((batch.length * 2) / 3)) // Delete 1/3rd of all of the batch elements
      this.logger.debug('Splicing batch...', {
        batchSizeInBytes: tx.length / 2,
      })
      tx = this.chainContract.interface.encodeFunctionData('appendStateBatch', [
        batch,
        startBlock,
      ])
    }

    this.logger.info('Generated state commitment batch', {
      batch, // list of stateRoots
    })
    return batch
  }
}
