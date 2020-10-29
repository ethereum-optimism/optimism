/* External Imports */
import { TransactionReceipt } from '@ethersproject/abstract-provider'
import { getContractFactory } from '@eth-optimism/contracts'

/* Internal Imports */
import { L2Block, Bytes32 } from '..'
import { RollupInfo, BatchSubmitter } from '.'

export class StateBatchSubmitter extends BatchSubmitter {
  // TODO: Change this so that we calculate start = scc.totalElements() and end = ctc.totalElements()!
  // Not based on the length of the L2 chain -- that is only used in the batch submitter
  // Note this means we've got to change the state / end calc logic

  protected l2ChainId: number
  protected syncing: boolean

  /*****************************
   * Batch Submitter Overrides *
   ****************************/

  public async _updateChainInfo(): Promise<void> {
    const info: RollupInfo = await this._getRollupInfo()
    if (info.mode === 'verifier') {
      throw new Error(
        'Verifier mode enabled! Batch submitter only compatible with sequencer mode'
      )
    }
    this.syncing = info.syncing
    const sccAddress = info.addresses.stateCommitmentChain

    if (
      typeof this.chainContract !== 'undefined' &&
      sccAddress === this.chainContract.address
    ) {
      return
    }

    this.chainContract = (
      await getContractFactory('OVM_StateCommitmentChain', this.signer)
    ).attach(sccAddress)

    this.log.info(
      `Initialized new State Commitment Chain with address: ${this.chainContract.address}`
    )
    return
  }

  public async _onSync(): Promise<TransactionReceipt> {
    this.log.info('Syncing mode enabled! Skipping state batch submission...')
    return
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
    if (tx.length < this.minTxSize) {
      this.log.info('State batch too small. Skipping batch submission...')
      return
    }
    return this._submitAndLogTx(
      this.chainContract.appendStateBatch(batch, startBlock),
      'Submitted state root batch!'
    )
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
      batch.push(block.stateRoot)
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
