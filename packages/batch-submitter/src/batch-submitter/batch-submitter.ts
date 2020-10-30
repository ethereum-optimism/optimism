/* External Imports */
import { Contract, Signer } from 'ethers'
import {
  TransactionResponse,
  TransactionReceipt,
} from '@ethersproject/abstract-provider'
import { Logger } from '@eth-optimism/core-utils'
import { OptimismProvider } from '@eth-optimism/provider'

/* Internal Imports */
import { Address, Bytes32 } from '../coders'

export interface RollupInfo {
  signer: Address
  mode: 'sequencer' | 'verifier'
  syncing: boolean
  l1BlockHash: Bytes32
  l1BlockHeight: number
  addresses: {
    canonicalTransactionChain: Address
    stateCommitmentChain: Address
    addressResolver: Address
    l1ToL2TransactionQueue: Address
    sequencerDecompression: Address
  }
}

export abstract class BatchSubmitter {
  protected rollupInfo: RollupInfo
  protected chainContract: Contract
  protected l2ChainId: number
  protected syncing: boolean

  constructor(
    readonly signer: Signer,
    readonly l2Provider: OptimismProvider,
    readonly minTxSize: number,
    readonly maxTxSize: number,
    readonly maxBatchSize: number,
    readonly numConfirmations: number,
    readonly log: Logger
  ) {}

  public abstract async _submitBatch(
    startBlock: number,
    endBlock: number
  ): Promise<TransactionReceipt>
  public abstract async _onSync(): Promise<TransactionReceipt>
  public abstract async _updateChainInfo(): Promise<void>

  public async submitNextBatch(): Promise<TransactionReceipt> {
    if (typeof this.l2ChainId === 'undefined') {
      this.l2ChainId = await this._getL2ChainId()
    }
    await this._updateChainInfo()

    if (this.syncing === true) {
      this.log.info(
        'Syncing mode enabled! Skipping batch submission and clearing queue...'
      )
      return this._onSync()
    }

    const startBlock =
      parseInt(await this.chainContract.getTotalElements(), 16) + 1 // +1 to skip L2 genesis block
    const endBlock = Math.min(
      startBlock + this.maxBatchSize,
      await this.l2Provider.getBlockNumber()
    )
    if (startBlock >= endBlock) {
      if (startBlock > endBlock) {
        this.log
          .error(`More chain elements in L1 (${startBlock}) than in the L2 node (${endBlock}).
                   This shouldn't happen because we don't submit batches if the sequencer is syncing.`)
      }
      this.log.info(`No txs to submit. Skipping batch submission...`)
      return
    }
    return this._submitBatch(startBlock, endBlock)
  }

  protected async _getRollupInfo(): Promise<RollupInfo> {
    return this.l2Provider.send('rollup_getInfo', [])
  }

  protected async _getL2ChainId(): Promise<number> {
    return this.l2Provider.send('eth_chainId', [])
  }

  protected async _submitAndLogTx(
    txPromise: Promise<TransactionResponse>,
    successMessage: string
  ): Promise<TransactionReceipt> {
    const response = await txPromise
    const receipt = await response.wait(this.numConfirmations)
    this.log.info(successMessage)
    this.log.debug('Transaction response:', response)
    this.log.debug('Transaction receipt:', receipt)
    return receipt
  }
}
