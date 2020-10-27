/* External Imports */
import { Contract, Signer } from 'ethers'
import {
  TransactionResponse,
  TransactionReceipt,
} from '@ethersproject/abstract-provider'
import { getLogger } from '@eth-optimism/core-utils'
import { OptimismProvider } from '@eth-optimism/provider'

/* Internal Imports */
import {
  Address,
  Bytes32,
} from '../coders'

/* Logging */
const log = getLogger('oe:batch-submitter:core')

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

export abstract class BatchSubmitter {
  protected rollupInfo: RollupInfo
  protected chainContract: Contract
  protected l2ChainId: number
  protected syncing: boolean

  constructor(
    readonly signer: Signer,
    readonly l2Provider: OptimismProvider,
    readonly maxTxSize: number,
    readonly maxBatchSize: number,
    readonly numConfirmations: number
  ) {}

  abstract async _submitBatch(startBlock: number, endBlock: number): Promise<TransactionReceipt>;
  abstract async _onSync(): Promise<TransactionReceipt>;
  abstract async _updateChainInfo(): Promise<void>;

  public async submitNextBatch(): Promise<TransactionReceipt> {
    if (typeof this.l2ChainId === 'undefined') {
      this.l2ChainId = await this._getL2ChainId()
    }
    await this._updateChainInfo()

    if (this.syncing === true) {
      log.info(
        'Syncing mode enabled! Skipping batch submission and clearing queue...'
      )
      return this._onSync()
    }

    const startBlock = parseInt(await this.chainContract.getTotalElements(), 16) + 1 // +1 to skip L2 genesis block
    const endBlock = Math.min(
      startBlock + this.maxBatchSize,
      await this.l2Provider.getBlockNumber()
    )
    if (startBlock >= endBlock) {
      if (startBlock > endBlock) {
        log.error(`More chain elements in L1 (${startBlock}) than in the L2 node (${endBlock}).
                   This shouldn't happen because we don't submit batches if the sequencer is syncing.`)
      }
      log.info(`No txs to submit. Skipping batch submission...`)
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
    log.info(successMessage)
    log.debug('Transaction response:', response)
    log.debug('Transaction receipt:', receipt)
    return receipt
  }
}