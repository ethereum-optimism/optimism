/* Imports: External */
import { fromHexString, sleep } from '@eth-optimism/core-utils'
import { BaseService, Metrics } from '@eth-optimism/common-ts'
import { TypedEvent } from '@eth-optimism/contracts/dist/types/common'
import { BaseProvider, StaticJsonRpcProvider } from '@ethersproject/providers'
import { LevelUp } from 'levelup'
import { BigNumber, constants } from 'ethers'
import { Gauge, Counter } from 'prom-client'

/* Imports: Internal */
import { handleEventsTransactionEnqueued } from './handlers/transaction-enqueued'
import { handleEventsSequencerBatchAppended } from './handlers/sequencer-batch-appended'
import { handleEventsStateBatchAppended } from './handlers/state-batch-appended'
import { MissingElementError } from './handlers/errors'
import { TransportDB } from '../../db/transport-db'
import {
  OptimismContracts,
  loadOptimismContracts,
  loadContract,
  validators,
} from '../../utils'
import { EventHandlerSet } from '../../types'
import { L1DataTransportServiceOptions } from '../main/service'

interface L1IngestionMetrics {
  highestSyncedL1Block: Gauge<string>
  missingElementCount: Counter<string>
  unhandledErrorCount: Counter<string>
}

const registerMetrics = ({
  client,
  registry,
}: Metrics): L1IngestionMetrics => ({
  highestSyncedL1Block: new client.Gauge({
    name: 'data_transport_layer_highest_synced_l1_block',
    help: 'Highest Synced L1 Block Number',
    registers: [registry],
  }),
  missingElementCount: new client.Counter({
    name: 'data_transport_layer_missing_element_count',
    help: 'Number of times recovery from missing elements happens',
    registers: [registry],
  }),
  unhandledErrorCount: new client.Counter({
    name: 'data_transport_layer_l1_unhandled_error_count',
    help: 'Number of times recovered from unhandled errors',
    registers: [registry],
  }),
})

export interface L1IngestionServiceOptions
  extends L1DataTransportServiceOptions {
  db: LevelUp
  metrics: Metrics
}

const optionSettings = {
  db: {
    validate: validators.isLevelUP,
  },
  addressManager: {
    validate: validators.isAddress,
  },
  confirmations: {
    default: 35,
    validate: validators.isInteger,
  },
  pollingInterval: {
    default: 5000,
    validate: validators.isInteger,
  },
  logsPerPollingInterval: {
    default: 2000,
    validate: validators.isInteger,
  },
  dangerouslyCatchAllErrors: {
    default: false,
    validate: validators.isBoolean,
  },
  l1RpcProvider: {
    validate: (val: any) => {
      return validators.isString(val) || validators.isJsonRpcProvider(val)
    },
  },
  l2ChainId: {
    validate: validators.isInteger,
  },
}

export class L1IngestionService extends BaseService<L1IngestionServiceOptions> {
  constructor(options: L1IngestionServiceOptions) {
    super('L1_Ingestion_Service', options, optionSettings)
  }

  private l1IngestionMetrics: L1IngestionMetrics

  private state: {
    db: TransportDB
    contracts: OptimismContracts
    l1RpcProvider: BaseProvider
    startingL1BlockNumber: number
    addressCache: {
      [name: string]: string
    }
  } = {} as any

  protected async _init(): Promise<void> {
    this.state.db = new TransportDB(this.options.db, {
      l2ChainId: this.options.l2ChainId,
    })

    this.l1IngestionMetrics = registerMetrics(this.metrics)

    if (typeof this.options.l1RpcProvider === 'string') {
      this.state.l1RpcProvider = new StaticJsonRpcProvider({
        url: this.options.l1RpcProvider,
        user: this.options.l1RpcProviderUser,
        password: this.options.l1RpcProviderPassword,
        headers: { 'User-Agent': 'data-transport-layer' },
      })
    } else {
      this.state.l1RpcProvider = this.options.l1RpcProvider
    }

    this.logger.info('Using AddressManager', {
      addressManager: this.options.addressManager,
    })

    const Lib_AddressManager = loadContract(
      'Lib_AddressManager',
      this.options.addressManager,
      this.state.l1RpcProvider
    )

    const code = await this.state.l1RpcProvider.getCode(
      Lib_AddressManager.address
    )
    if (fromHexString(code).length === 0) {
      throw new Error(
        `Provided AddressManager doesn't have any code: ${Lib_AddressManager.address}`
      )
    }

    try {
      // Just check to make sure this doesn't throw. If this is a valid AddressManager, then this
      // call should succeed. If it throws, then our AddressManager is broken. We don't care about
      // the result.
      await Lib_AddressManager.getAddress(
        `Here's a contract name that definitely doesn't exist.`
      )
    } catch (err) {
      throw new Error(
        `Seems like your AddressManager is busted: ${Lib_AddressManager.address}`
      )
    }

    // Would be nice if this weren't necessary, maybe one day.
    // TODO: Probably just assert inside here that all of the contracts have code in them.
    this.state.contracts = await loadOptimismContracts(
      this.state.l1RpcProvider,
      this.options.addressManager
    )

    // Look up in the database for an indexed starting L1 block
    let startingL1BlockNumber = await this.state.db.getStartingL1Block()
    // If there isn't an indexed starting L1 block, that means we should pull it
    // from config and then fallback to discovering it
    if (startingL1BlockNumber === null || startingL1BlockNumber === undefined) {
      if (
        this.options.l1StartHeight !== null &&
        this.options.l1StartHeight !== undefined
      ) {
        startingL1BlockNumber = this.options.l1StartHeight
      } else {
        this.logger.info(
          'Attempting to find an appropriate L1 block height to begin sync. This may take a long time.'
        )
        startingL1BlockNumber = await this._findStartingL1BlockNumber()
      }
    }

    if (!startingL1BlockNumber) {
      throw new Error('Cannot find starting L1 block number')
    }

    this.logger.info('Starting sync', {
      startingL1BlockNumber,
    })

    this.state.startingL1BlockNumber = startingL1BlockNumber
    await this.state.db.setStartingL1Block(this.state.startingL1BlockNumber)

    // Store the total number of submitted transactions so the server can tell clients if we're
    // done syncing or not
    const totalElements =
      await this.state.contracts.CanonicalTransactionChain.getTotalElements()
    if (totalElements > 0) {
      await this.state.db.putHighestL2BlockNumber(totalElements - 1)
    }

    // Initialize the address cache.
    this.state.addressCache = {}
  }

  protected async _start(): Promise<void> {
    // This is our main function. It's basically just an infinite loop that attempts to stay in
    // sync with events coming from Ethereum. Loops as quickly as it can until it approaches the
    // tip of the chain, after which it starts waiting for a few seconds between each loop to avoid
    // unnecessary spam.
    while (this.running) {
      try {
        const highestSyncedL1Block =
          (await this.state.db.getHighestSyncedL1Block()) ||
          this.state.startingL1BlockNumber
        const currentL1Block = await this.state.l1RpcProvider.getBlockNumber()
        const targetL1Block = Math.min(
          highestSyncedL1Block + this.options.logsPerPollingInterval,
          currentL1Block - this.options.confirmations
        )

        // We're already at the head, so no point in attempting to sync.
        if (highestSyncedL1Block === targetL1Block) {
          await sleep(this.options.pollingInterval)
          continue
        }

        this.logger.info('Synchronizing events from Layer 1 (Ethereum)', {
          highestSyncedL1Block,
          targetL1Block,
        })

        // We always try loading the deposit shutoff block in every loop! Less efficient but will
        // guarantee that we don't miss the shutoff block being set and that we can automatically
        // recover if the shutoff block is set and then unset.
        const depositShutoffBlock = BigNumber.from(
          await this.state.contracts.Lib_AddressManager.getAddress(
            'DTL_SHUTOFF_BLOCK'
          )
        ).toNumber()

        // If the deposit shutoff block is set, then we should stop syncing deposits at that block.
        let depositTargetL1Block = targetL1Block
        if (depositShutoffBlock > 0) {
          this.logger.info(`Deposit shutoff active`, {
            targetL1Block,
            depositShutoffBlock,
          })
          depositTargetL1Block = Math.min(
            depositTargetL1Block,
            depositShutoffBlock
          )
        }

        // I prefer to do this in serial to avoid non-determinism. We could have a discussion about
        // using Promise.all if necessary, but I don't see a good reason to do so unless parsing is
        // really, really slow for all event types.
        await this._syncEvents(
          'CanonicalTransactionChain',
          'TransactionEnqueued',
          highestSyncedL1Block,
          depositTargetL1Block,
          handleEventsTransactionEnqueued
        )

        await this._syncEvents(
          'CanonicalTransactionChain',
          'SequencerBatchAppended',
          highestSyncedL1Block,
          targetL1Block,
          handleEventsSequencerBatchAppended
        )

        await this._syncEvents(
          'StateCommitmentChain',
          'StateBatchAppended',
          highestSyncedL1Block,
          targetL1Block,
          handleEventsStateBatchAppended
        )

        await this.state.db.setHighestSyncedL1Block(targetL1Block)

        this.l1IngestionMetrics.highestSyncedL1Block.set(targetL1Block)

        if (
          currentL1Block - highestSyncedL1Block <
          this.options.logsPerPollingInterval
        ) {
          await sleep(this.options.pollingInterval)
        }
      } catch (err) {
        if (err instanceof MissingElementError) {
          this.logger.warn('recovering from a missing event', {
            message: err.toString(),
          })

          // Different functions for getting the last good element depending on the event type.
          const handlers = {
            SequencerBatchAppended:
              this.state.db.getLatestTransactionBatch.bind(this.state.db),
            SequencerBatchAppendedTransaction:
              this.state.db.getLatestTransaction.bind(this.state.db),
            StateBatchAppended: this.state.db.getLatestStateRootBatch.bind(
              this.state.db
            ),
            TransactionEnqueued: this.state.db.getLatestEnqueue.bind(
              this.state.db
            ),
          }

          // Find the last good element and reset the highest synced L1 block to go back to the
          // last good element. Will resync other event types too but we have no issues with
          // syncing the same events more than once.
          const eventName = err.name
          if (!(eventName in handlers)) {
            throw new Error(
              `unable to recover from missing event, no handler for ${eventName}`
            )
          }

          const lastGoodElement: {
            blockNumber: number
          } = await handlers[eventName]()

          // Erroring out here seems fine. An error like this is only likely to occur quickly after
          // this service starts up so someone will be here to deal with it. Automatic recovery is
          // nice but not strictly necessary. Could be a good feature for someone to implement.
          if (lastGoodElement === null) {
            throw new Error(`unable to recover from missing event`)
          }

          // Rewind back to the block number that the last good element was in.
          await this.state.db.setHighestSyncedL1Block(
            lastGoodElement.blockNumber
          )

          this.l1IngestionMetrics.highestSyncedL1Block.set(
            lastGoodElement.blockNumber
          )

          // Something we should be keeping track of.
          this.logger.warn('recovered from a missing event', {
            eventName,
            lastGoodBlockNumber: lastGoodElement.blockNumber,
          })

          this.l1IngestionMetrics.missingElementCount.inc()
        } else if (!this.running || this.options.dangerouslyCatchAllErrors) {
          this.l1IngestionMetrics.unhandledErrorCount.inc()
          this.logger.error('Caught an unhandled error', {
            message: err.toString(),
            stack: err.stack,
            code: err.code,
          })
          await sleep(this.options.pollingInterval)
        } else {
          throw err
        }
      }
    }
  }

  private async _syncEvents(
    contractName: string,
    eventName: string,
    fromL1Block: number,
    toL1Block: number,
    handlers: EventHandlerSet<any, any, any>
  ): Promise<void> {
    // Basic sanity checks.
    if (!this.state.contracts[contractName]) {
      throw new Error(`Contract ${contractName} does not exist.`)
    }

    // Basic sanity checks.
    if (!this.state.contracts[contractName].filters[eventName]) {
      throw new Error(
        `Event ${eventName} does not exist on contract ${contractName}`
      )
    }

    // We need to figure out how to make this work without Infura. Mark and I think that infura is
    // doing some indexing of events beyond Geth's native capabilities, meaning some event logic
    // will only work on Infura and not on a local geth instance. Not great.
    const addressSetEvents =
      await this.state.contracts.Lib_AddressManager.queryFilter(
        this.state.contracts.Lib_AddressManager.filters.AddressSet(
          contractName
        ),
        fromL1Block,
        toL1Block
      )

    // We're going to parse things out in ranges because the address of a given contract may have
    // changed in the range provided by the user.
    const eventRanges: {
      address: string
      fromBlock: number
      toBlock: number
    }[] = []

    let l1BlockRangeStart = fromL1Block

    // Addresses on mainnet do NOT change. We can therefore skip costly checks related to cases
    // where addresses do change (mainly on testnets).
    if (this.options.l2ChainId === 10) {
      eventRanges.push({
        address: await this._getFixedAddress(contractName),
        fromBlock: l1BlockRangeStart,
        toBlock: toL1Block,
      })
    } else {
      // Addresses can change on non-mainnet deployments. If an address changes, we will
      // potentially need to sync events from both the old address and the new address. We will
      // add an entry here for each address change.
      for (const addressSetEvent of addressSetEvents) {
        eventRanges.push({
          address: addressSetEvent.args._oldAddress,
          fromBlock: l1BlockRangeStart,
          toBlock: addressSetEvent.blockNumber,
        })

        l1BlockRangeStart = addressSetEvent.blockNumber
      }

      // Add one more range to get us to the end of the user-provided block range.
      eventRanges.push({
        address: await this._getContractAddressAtBlock(contractName, toL1Block),
        fromBlock: l1BlockRangeStart,
        toBlock: toL1Block,
      })
    }

    for (const eventRange of eventRanges) {
      // Find all relevant events within the range.
      const events: TypedEvent[] = await this.state.contracts[contractName]
        .attach(eventRange.address)
        .queryFilter(
          this.state.contracts[contractName].filters[eventName](),
          eventRange.fromBlock,
          eventRange.toBlock
        )

      // Handle events, if any.
      if (events.length > 0) {
        const tick = Date.now()

        for (const event of events) {
          const extraData = await handlers.getExtraData(
            event,
            this.state.l1RpcProvider
          )
          const parsedEvent = await handlers.parseEvent(
            event,
            extraData,
            this.options.l2ChainId
          )
          await handlers.storeEvent(parsedEvent, this.state.db)
        }

        const tock = Date.now()

        this.logger.info('Processed events', {
          eventName,
          numEvents: events.length,
          durationMs: tock - tick,
        })
      }
    }
  }

  /**
   * Gets the address of a contract at a particular block in the past.
   *
   * @param contractName Name of the contract to get an address for.
   * @param blockNumber Block at which to get an address.
   * @return Contract address.
   */
  private async _getContractAddressAtBlock(
    contractName: string,
    blockNumber: number
  ): Promise<string> {
    const events = await this.state.contracts.Lib_AddressManager.queryFilter(
      this.state.contracts.Lib_AddressManager.filters.AddressSet(contractName),
      this.state.startingL1BlockNumber,
      blockNumber
    )

    if (events.length > 0) {
      return events[events.length - 1].args._newAddress
    } else {
      // Address wasn't set before this.
      return constants.AddressZero
    }
  }

  /**
   * Returns an address for a contract name whose address isn't expected to change.
   *
   * @param contractName Name of the contract to get an address for.
   * @return Contract address.
   */
  private async _getFixedAddress(contractName: string): Promise<string> {
    if (this.state.addressCache[contractName]) {
      return this.state.addressCache[contractName]
    }

    const address =
      this.state.contracts.Lib_AddressManager.getAddress(contractName)
    this.state.addressCache[contractName] = address
    return address
  }

  private async _findStartingL1BlockNumber(): Promise<number> {
    const currentL1Block = await this.state.l1RpcProvider.getBlockNumber()
    const filter =
      this.state.contracts.Lib_AddressManager.filters.OwnershipTransferred()

    for (let i = 0; i < currentL1Block; i += 2000) {
      const start = i
      const end = Math.min(i + 2000, currentL1Block)
      this.logger.info(`Searching for ${filter} from ${start} to ${end}`)

      const events = await this.state.contracts.Lib_AddressManager.queryFilter(
        filter,
        start,
        end
      )

      if (events.length > 0) {
        return events[0].blockNumber
      }
    }

    throw new Error(`Unable to find appropriate L1 starting block number`)
  }
}
