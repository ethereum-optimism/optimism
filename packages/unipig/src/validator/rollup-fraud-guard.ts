/* External Imports */
import { DB, EthereumEvent, EthereumListener } from '@pigi/core-db'
import { getLogger, logError } from '@pigi/core-utils'
import * as AsyncLock from 'async-lock'

import { Contract } from 'ethers'
import { TransactionReceipt } from 'ethers/providers'

/* Internal Imports */
import { parseTransitionFromABI } from '../common/serialization'
import { RollupBlock, RollupStateValidator } from '../types'
import { ContractFraudProof } from './types'

const log = getLogger('rollup-fraud-guard')

/**
 * Handles NewRollupBlock events, checks for fraud, submits proof when there is fraud.
 */
export class RollupFraudGuard implements EthereumListener<EthereumEvent> {
  public static readonly LAST_BLOCK_VALIDATED_KEY = Buffer.from(
    'LAST_VALIDATED_BLOCK'
  )
  private static readonly lockKey: string = 'lock'

  private readonly lock: AsyncLock
  private lastBlockValidated: number

  public static async create(
    db: DB,
    validator: RollupStateValidator,
    contract: Contract
  ): Promise<RollupFraudGuard> {
    const fraudGuard: RollupFraudGuard = new RollupFraudGuard(
      db,
      validator,
      contract
    )

    await fraudGuard.init()

    return fraudGuard
  }

  private constructor(
    private readonly db: DB,
    private readonly validator: RollupStateValidator,
    private readonly contract: Contract
  ) {
    this.lock = new AsyncLock()
  }

  private async init(): Promise<void> {
    const lastValidatedBuffer: Buffer = await this.db.get(
      RollupFraudGuard.LAST_BLOCK_VALIDATED_KEY
    )

    if (!lastValidatedBuffer) {
      log.info(`Starting fresh. Last validated block: 0`)
      this.lastBlockValidated = 0
      return
    }

    this.lastBlockValidated = parseInt(lastValidatedBuffer.toString(), 10)
    log.info(
      `Starting from previous run. Last validated block: ${this.lastBlockValidated}`
    )
  }

  public async onSyncCompleted(syncIdentifier?: string): Promise<void> {
    log.info(`Synced with rollup chain. Awaiting new events to validate...`)
  }

  public async handle(event: EthereumEvent): Promise<void> {
    log.debug(`Fraud Guard received event: ${JSON.stringify(event)}`)
    if (
      !event ||
      (!event.values &&
        !('block' in event.values) &&
        !('blockNumber' in event.values))
    ) {
      log.error(`Unrecognized event. Returning`)
      return
    }

    let block: RollupBlock
    try {
      block = {
        blockNumber: (event.values['blockNumber'] as any).toNumber(),
        transitions: (event.values['block'] as string[]).map((x) =>
          parseTransitionFromABI(x)
        ),
      }

      return this.lock.acquire(RollupFraudGuard.lockKey, async () => {
        await this.handleNewRollupBlock(block)
      })
    } catch (e) {
      logError(
        log,
        `Error trying to parsing and handling event: ${JSON.stringify(event)}`,
        e
      )
      return
    }
  }

  private async handleNewRollupBlock(block: RollupBlock): Promise<void> {
    if (this.lastBlockValidated >= block.blockNumber) {
      log.debug(
        `Received event for old block. Ignoring. lastValidated: ${
          this.lastBlockValidated
        }. Received: ${JSON.stringify(block)}`
      )
      return
    } else if (this.lastBlockValidated + 1 !== block.blockNumber) {
      log.error(
        `Received event with block number greater than expected! lastValidated: ${
          this.lastBlockValidated
        }. Received: ${JSON.stringify(block)}`
      )
      process.exit(1)
    }

    await this.validator.storeBlock(block)
    let proof: ContractFraudProof
    try {
      proof = await this.validator.validateStoredBlock(block.blockNumber)
    } catch (e) {
      logError(log, `Error validating block: ${JSON.stringify(block)}`, e)
      process.exit(1)
    }

    if (!!proof) {
      await this.submitFraudProof(proof)
    } else {
      log.debug(`Validated that block ${block.blockNumber} is not fraudulent.`)
      this.lastBlockValidated = block.blockNumber
      await this.db.put(
        RollupFraudGuard.LAST_BLOCK_VALIDATED_KEY,
        Buffer.from(this.lastBlockValidated.toString(10))
      )
    }
  }

  private async submitFraudProof(proof: ContractFraudProof): Promise<void> {
    log.error(
      `Detected fraud. Submitting Fraud Proof: ${JSON.stringify(proof)}`
    )

    try {
      const receipt: TransactionReceipt = await this.contract.proveTransitionInvalid(
        ...proof
      )
      log.error(`Fraud proof submitted. Receipt: ${JSON.stringify(receipt)}`)
    } catch (e) {
      logError(log, 'Error submitting fraud proof!', e)
      process.exit(1)
    }

    log.info('Congrats! You helped the good guys win. +2 points for you!')
    process.exit(0)
  }
}
