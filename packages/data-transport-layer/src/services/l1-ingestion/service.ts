/* Imports: External */
import { fromHexString, EventArgsAddressSet } from '@eth-optimism/core-utils'
import { BaseService } from '@eth-optimism/common-ts'
import { JsonRpcProvider } from '@ethersproject/providers'
import { LevelUp } from 'levelup'
import { ethers, constants } from 'ethers'

/* Imports: Internal */
import { TransportDB } from '../../db/transport-db'
import {
  OptimismContracts,
  sleep,
  loadOptimismContracts,
  loadContract,
  validators,
} from '../../utils'
import { TypedEthersEvent, EventHandlerSet } from '../../types'
import { handleEventsTransactionEnqueued } from './handlers/transaction-enqueued'
import { handleEventsSequencerBatchAppended } from './handlers/sequencer-batch-appended'
import { handleEventsStateBatchAppended } from './handlers/state-batch-appended'
import { L1DataTransportServiceOptions } from '../main/service'

export interface L1IngestionServiceOptions
  extends L1DataTransportServiceOptions {
  db: LevelUp
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
      return validators.isUrl(val) || validators.isJsonRpcProvider(val)
    },
  },
}

export class L1IngestionService extends BaseService<L1IngestionServiceOptions> {
  constructor(options: L1IngestionServiceOptions) {
    super('L1_Ingestion_Service', options, optionSettings)
  }

  private state: {
    db: TransportDB
    contracts: OptimismContracts
    l1RpcProvider: JsonRpcProvider
    startingL1BlockNumber: number
    l2ChainId: number
  } = {} as any

  protected async _init(): Promise<void> {
    this.state.db = new TransportDB(this.options.db)

    this.state.l1RpcProvider =
      typeof this.options.l1RpcProvider === 'string'
        ? new JsonRpcProvider(this.options.l1RpcProvider)
        : this.options.l1RpcProvider

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

    this.state.l2ChainId = ethers.BigNumber.from(
      await this.state.contracts.OVM_ExecutionManager.ovmCHAINID()
    ).toNumber()

    const startingL1BlockNumber = await this.state.db.getStartingL1Block()
    if (startingL1BlockNumber) {
      this.state.startingL1BlockNumber = startingL1BlockNumber
    } else {
      this.logger.info(
        'Attempting to find an appropriate L1 block height to begin sync...'
      )
      this.state.startingL1BlockNumber = await this._findStartingL1BlockNumber()
      this.logger.info('Starting sync', {
        startingL1BlockNumber: this.state.startingL1BlockNumber,
      })

      await this.state.db.setStartingL1Block(this.state.startingL1BlockNumber)
    }

    // Store the total number of submitted transactions so the server can tell clients if we're
    // done syncing or not
    const totalElements = await this.state.contracts.OVM_CanonicalTransactionChain.getTotalElements()
    if (totalElements > 0) {
      await this.state.db.putHighestL2BlockNumber(totalElements - 1)
    }
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

        // I prefer to do this in serial to avoid non-determinism. We could have a discussion about
        // using Promise.all if necessary, but I don't see a good reason to do so unless parsing is
        // really, really slow for all event types.
        await this._syncEvents(
          'OVM_CanonicalTransactionChain',
          'TransactionEnqueued',
          highestSyncedL1Block,
          targetL1Block,
          handleEventsTransactionEnqueued
        )

        await this._syncEvents(
          'OVM_CanonicalTransactionChain',
          'SequencerBatchAppended',
          highestSyncedL1Block,
          targetL1Block,
          handleEventsSequencerBatchAppended
        )

        await this._syncEvents(
          'OVM_StateCommitmentChain',
          'StateBatchAppended',
          highestSyncedL1Block,
          targetL1Block,
          handleEventsStateBatchAppended
        )

        await this.state.db.setHighestSyncedL1Block(targetL1Block)

        if (
          currentL1Block - highestSyncedL1Block <
          this.options.logsPerPollingInterval
        ) {
          await sleep(this.options.pollingInterval)
        }
      } catch (err) {
        if (!this.running || this.options.dangerouslyCatchAllErrors) {
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
    const addressSetEvents = ((await this.state.contracts.Lib_AddressManager.queryFilter(
      this.state.contracts.Lib_AddressManager.filters.AddressSet(),
      fromL1Block,
      toL1Block
    )) as TypedEthersEvent<EventArgsAddressSet>[]).filter((event) => {
      return event.args._name === contractName
    })

    // We're going to parse things out in ranges because the address of a given contract may have
    // changed in the range provided by the user.
    const eventRanges: {
      address: string
      fromBlock: number
      toBlock: number
    }[] = []

    // Add a range for each address change.
    let l1BlockRangeStart = fromL1Block
    for (const addressSetEvent of addressSetEvents) {
      eventRanges.push({
        address: await this._getContractAddressAtBlock(
          contractName,
          addressSetEvent.blockNumber
        ),
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

    for (const eventRange of eventRanges) {
      // Find all relevant events within the range.
      const events: TypedEthersEvent<any>[] = await this.state.contracts[
        contractName
      ]
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
            this.state.l2ChainId
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
   * @param contractName Name of the contract to get an address for.
   * @param blockNumber Block at which to get an address.
   * @return Contract address.
   */
  private async _getContractAddressAtBlock(
    contractName: string,
    blockNumber: number
  ): Promise<string> {
    // TODO: Should be much easier than this. Need to change the params of this event.
    const relevantAddressSetEvents = (
      await this.state.contracts.Lib_AddressManager.queryFilter(
        this.state.contracts.Lib_AddressManager.filters.AddressSet(),
        this.state.startingL1BlockNumber
      )
    ).filter((event) => {
      return (
        event.args._name === contractName && event.blockNumber < blockNumber
      )
    })

    if (relevantAddressSetEvents.length > 0) {
      return relevantAddressSetEvents[relevantAddressSetEvents.length - 1].args
        ._newAddress
    } else {
      // Address wasn't set before this.
      return constants.AddressZero
    }
  }

  private async _findStartingL1BlockNumber(): Promise<number> {
    const currentL1Block = await this.state.l1RpcProvider.getBlockNumber()

    for (let i = 0; i < currentL1Block; i += 1000000) {
      const events = await this.state.contracts.Lib_AddressManager.queryFilter(
        this.state.contracts.Lib_AddressManager.filters.AddressSet(),
        i,
        Math.min(i + 1000000, currentL1Block)
      )

      if (events.length > 0) {
        return events[0].blockNumber
      }
    }

    throw new Error(`Unable to find appropriate L1 starting block number`)
  }
}
