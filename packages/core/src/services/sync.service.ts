/* External Imports */
import { Service, OnStart } from '@nestd/core'
import { sleep, Transaction } from '@pigi/utils'

/* Services */
import { LoggerService, SyncLogger } from './logging'
import { EventService } from './event.service'
import { SyncDB } from './db/interfaces/sync-db'
import { ChainDB } from './db/interfaces/chain-db'
import { ChainService } from './chain.service'
import { OperatorService } from './operator.service'
import { ContractService } from './eth/contract.service'
import { ConfigService } from './config.service'
import { WalletService } from './wallet.service'

/* Internal Imports */
import {
  BlockSubmittedEvent,
  DepositEvent,
  ExitFinalizedEvent,
  ExitStartedEvent,
} from '../models/events'
import { CONFIG } from '../constants'

interface SyncServiceOptions {
  transactionPollInterval: number
}

/**
 * Service used to synchronize the local database.
 */
@Service()
export class SyncService implements OnStart {
  private readonly logger = new SyncLogger('sync', this.logs)
  private pending: Transaction[] = []
  private polling = false

  constructor(
    private readonly logs: LoggerService,
    private readonly events: EventService,
    private readonly syncdb: SyncDB,
    private readonly chaindb: ChainDB,
    private readonly chain: ChainService,
    private readonly operator: OperatorService,
    private readonly contract: ContractService,
    private readonly wallet: WalletService,
    private readonly config: ConfigService
  ) {}

  public async onStart(): Promise<void> {
    this.attachHandlers()
  }

  /**
   * Starts regularly polling pending transactions.
   */
  public async startPollInterval(): Promise<void> {
    if (this.polling) {
      return
    }

    this.polling = true
    this.pollInterval()
  }

  private options(): SyncServiceOptions {
    return this.config.get(CONFIG.SYNC_SERVICE_OPTIONS)
  }

  /**
   * Polling loop that checks for new transactions.
   */
  private async pollInterval(): Promise<void> {
    try {
      await this.checkPendingTransactions()
    } finally {
      await sleep(this.options().transactionPollInterval)
      this.pollInterval()
    }
  }

  /**
   * Attaches handlers to Ethereum events.
   */
  private attachHandlers(): void {
    const handlers: { [key: string]: (events: any[]) => void } = {
      BlockSubmitted: this.onBlockSubmitted,
      Deposit: this.onDeposit,
      ExitFinalized: this.onExitFinalized,
      ExitStarted: this.onExitStarted,
    }

    for (const event of Object.keys(handlers)) {
      this.events.on(`eventHandler.${event}`, handlers[event].bind(this))
    }
  }

  /**
   * Checks for any available pending transactions and emits an event for each.
   */
  private async checkPendingTransactions() {
    if (!this.operator.isConnected() || !this.contract.hasAddress) {
      return
    }

    const lastSyncedBlock = await this.syncdb.getLastSyncedBlock()
    const firstUnsyncedBlock = lastSyncedBlock + 1
    const currentBlock = await this.chaindb.getLatestBlock()
    const prevFailed = await this.syncdb.getFailedTransactions()

    if (firstUnsyncedBlock <= currentBlock) {
      this.logger.log(
        `Checking for new transactions between plasma blocks ${firstUnsyncedBlock} and ${currentBlock}.`
      )
    } else if (prevFailed.length > 0) {
      this.logger.log(`Attempting to apply failed transactions.`)
    } else {
      return
    }

    // TODO: Figure out how handle operator errors.
    const addresses = await this.wallet.getAccounts()
    for (const address of addresses) {
      const received = await this.operator.getReceivedTransactions(
        address,
        firstUnsyncedBlock,
        currentBlock
      )
      this.pending = this.pending.concat(received)
    }

    // Add any previously failed transactions to try again.
    this.pending = this.pending.concat(prevFailed)

    // Remove any duplicates
    this.pending = Array.from(new Set(this.pending))

    const failed = []
    for (const transaction of this.pending) {
      // Make sure we're not importing transactions we don't have blocks for.
      // Necessary because of a bug in the operator.
      // TODO: Fix operator so this isn't necessary.
      if (transaction.block.gtn(currentBlock)) {
        continue
      }

      try {
        await this.addTransaction(transaction)
      } catch (err) {
        failed.push(transaction)
        this.logger.error('Could not import transaction', err)
        this.logger.log(
          `Ran into an error while importing transaction: ${
            transaction.hash
          }, trying again in a few seconds...`
        )
      }
    }

    await this.syncdb.setFailedTransactions(failed)
    await this.syncdb.setLastSyncedBlock(currentBlock)
  }

  /**
   * Tries to add any newly received transactions.
   * @param tx A signed transaction.
   */
  private async addTransaction(tx: Transaction) {
    if (await this.chaindb.hasTransaction(tx.hash)) {
      return
    }

    this.logger.log(`Detected new transaction: ${tx.hash}`)
    this.logger.log(`Attemping to pull information for transaction: ${tx.hash}`)
    let proof
    try {
      proof = await this.operator.getTransactionProof(tx.encoded)
    } catch (err) {
      this.logger.error(
        `Operator failed to return information for transaction: ${tx.hash}`,
        err
      )
      throw err
    }

    this.logger.log(`Importing new transaction: ${tx.hash}`)
    await this.chain.addTransaction(proof)
    this.logger.log(`Successfully imported transaction: ${tx.hash}`)
  }

  /**
   * Handles new deposit events.
   * @param deposits Deposit events.
   */
  private async onDeposit(events: DepositEvent[]): Promise<void> {
    const deposits = events.map((event) => {
      return event.toDeposit()
    })
    await this.chain.addDeposits(deposits)
  }

  /**
   * Handles new block events.
   * @param blocks Block submission events.
   */
  private async onBlockSubmitted(events: BlockSubmittedEvent[]): Promise<void> {
    const blocks = events.map((event) => {
      return event.toBlock()
    })
    await this.chaindb.addBlockHeaders(blocks)
  }

  /**
   * Handles new exit started events.
   * @param exits Exit started events.
   */
  private async onExitStarted(events: ExitStartedEvent[]): Promise<void> {
    const exits = events.map((event) => {
      return event.toExit()
    })
    for (const exit of exits) {
      await this.chain.addExit(exit)
    }
  }

  /**
   * Handles new exit finalized events.
   * @param exits Exit finalized events.
   */
  private async onExitFinalized(exits: ExitFinalizedEvent[]): Promise<void> {
    for (const exit of exits) {
      await this.chaindb.markFinalized(exit)
      await this.chaindb.addExitableEnd(exit.start)
    }
  }
}
