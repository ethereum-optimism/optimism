/* External Imports */
import { Contract, Signer } from 'ethers'
import { TransactionReceipt } from '@ethersproject/abstract-provider'
import { getContractFactory } from '@eth-optimism/contracts'
import { Logger, Bytes32 } from '@eth-optimism/core-utils'
import { OptimismProvider } from '@eth-optimism/provider'

/* Internal Imports */
import { L2Block } from '..'
import { RollupInfo, Range, BatchSubmitter, BLOCK_OFFSET } from '.'

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
    l2Provider: OptimismProvider,
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
    log: Logger,
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
      log
    )
    this.fraudSubmissionAddress = fraudSubmissionAddress
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
    const addrs = await this._getChainAddresses()
    const sccAddress = addrs.sccAddress
    const ctcAddress = addrs.ctcAddress

    if (
      typeof this.chainContract !== 'undefined' &&
      sccAddress === this.chainContract.address &&
      ctcAddress === this.ctcContract.address
    ) {
      return
    }

    this.chainContract = (
      await getContractFactory('OVM_StateCommitmentChain', this.signer)
    ).attach(sccAddress)
    this.ctcContract = (
      await getContractFactory('OVM_CanonicalTransactionChain', this.signer)
    ).attach(ctcAddress)

    this.log.info('Connected Optimism contracts', {
      stateCommitmentChain: this.chainContract.address,
      canonicalTransactionChain: this.ctcContract.address,
    })
    return
  }

  public async _onSync(): Promise<TransactionReceipt> {
    this.log.info('Syncing mode enabled! Skipping state batch submission...')
    return
  }

  public async _getBatchStartAndEnd(): Promise<Range> {
    // TODO: Remove BLOCK_OFFSET by adding a tx to Geth's genesis
    const startBlock: number =
      (await this.chainContract.getTotalElements()).toNumber() + BLOCK_OFFSET
    // We will submit state roots for txs which have been in the tx chain for a while.
    const callBlockNumber: number =
      (await this.signer.provider.getBlockNumber()) - this.finalityConfirmations
    const totalElements: number =
      (await this.ctcContract.getTotalElements()).toNumber() + BLOCK_OFFSET
    const endBlock: number = Math.min(
      startBlock + this.maxBatchSize,
      totalElements
    )
    if (startBlock >= endBlock) {
      if (startBlock > endBlock) {
        this.log.error(
          'State commitment chain is larger than transaction chain. This should never happen!'
        )
      }
      this.log.info(
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
    if (!this._shouldSubmitBatch(tx.length * 2)) {
      return
    }

    const offsetStartsAtIndex = startBlock - BLOCK_OFFSET // TODO: Remove BLOCK_OFFSET by adding a tx to Geth's genesis
    this.log.debug('Submitting batch.', { tx })

    const nonce = await this.signer.getTransactionCount()
    const contractFunction = async (gasPrice): Promise<TransactionReceipt> => {
      const contractTx = await this.chainContract.appendStateBatch(
        batch,
        offsetStartsAtIndex,
        { nonce, gasPrice }
      )
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
    const batch: Bytes32[] = []

    for (let i = startBlock; i < endBlock; i++) {
      const block = (await this.l2Provider.getBlockWithTransactions(
        i
      )) as L2Block
      if (block.transactions[0].from === this.fraudSubmissionAddress) {
        batch.push(
          '0xbad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1bad1'
        )
        this.fraudSubmissionAddress = 'no fraud'
      } else {
        batch.push(block.stateRoot)
      }
    }

    let tx = this.chainContract.interface.encodeFunctionData(
      'appendStateBatch',
      [batch, startBlock]
    )
    while (tx.length > this.maxTxSize) {
      batch.splice(Math.ceil((batch.length * 2) / 3)) // Delete 1/3rd of all of the batch elements
      tx = this.chainContract.interface.encodeFunctionData('appendStateBatch', [
        batch,
        startBlock,
      ])
    }
    return batch
  }
}
