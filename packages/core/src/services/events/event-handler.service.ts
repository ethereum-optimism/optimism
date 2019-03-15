/* External Imports */
import { Service, OnStart } from '@nestd/core'

/* Services */
import { EventWatcherService } from './event-watcher.service'
import { EventService } from '../event.service'
import { LoggerService, SyncLogger } from '../logging'

/* Internal Imports */
import { EthereumEvent } from '../../models/eth'
import { PlasmaBlock, Deposit, Exit } from '../../models/chain'

/**
 * Service that handles events coming from
 * EventWatcherService and parses them into
 * useable objects.
 */
@Service()
export class EventHandlerService implements OnStart {
  private readonly name = 'eventHandler'
  private readonly logger = new SyncLogger(this.name, this.logs)

  constructor(
    private readonly logs: LoggerService,
    private readonly events: EventService,
    private readonly eventWatcher: EventWatcherService
  ) {}

  public async onStart(): Promise<void> {
    this.registerHandlers()
  }

  /**
   * Emits a prefixed event.
   * @param event Name of the event.
   * @param args Event object.
   */
  private emitContractEvent(event: string, ...args: any): void {
    this.events.event(this.name, event, args)
  }

  /**
   * Registers event handlers.
   */
  private registerHandlers(): void {
    const handlers: { [key: string]: (...args: any) => any } = {
      BeginExitEvent: this.onExitStarted,
      DepositEvent: this.onDeposit,
      FinalizeExitEvent: this.onExitFinalized,
      SubmitBlockEvent: this.onBlockSubmitted,
    }
    for (const event of Object.keys(handlers)) {
      this.eventWatcher.subscribe(event)
      this.events.on(`eventWatcher.${event}`, handlers[event].bind(this))
    }
  }

  /**
   * Handles Deposit events.
   * @param events Deposit events.
   */
  private onDeposit(events: EthereumEvent[]): void {
    const deposits = events.map(Deposit.from)
    deposits.forEach((deposit) => {
      this.logger.log(
        `Detected new deposit of ${deposit.end.sub(deposit.start)} for ${
          deposit.owner
        }`
      )
    })
    this.emitContractEvent('Deposit', deposits)
  }

  /**
   * Handles BlockSubmitted events.
   * @param events BlockSubmitted events.
   */
  private onBlockSubmitted(events: EthereumEvent[]): void {
    const blocks = events.map(PlasmaBlock.from)
    blocks.forEach((block) => {
      this.logger.log(`Detected block #${block.number}: ${block.hash}`)
    })
    this.emitContractEvent('BlockSubmitted', blocks)
  }

  /**
   * Handles ExitStarted events.
   * @param events ExitStarted events.
   */
  private onExitStarted(events: EthereumEvent[]): void {
    const exits = events.map(Exit.from)
    exits.forEach((exit) => {
      this.logger.log(`Detected new started exit: ${exit.id}`)
    })
    this.emitContractEvent('ExitStarted', exits)
  }

  /**
   * Handles ExitFinalized events.
   * @param events ExitFinalized events.
   */
  private onExitFinalized(events: EthereumEvent[]): void {
    const exits = events.map(Exit.from)
    exits.forEach((exit) => {
      this.logger.log(`Detected new finalized exit: ${exit.id}`)
    })
    this.emitContractEvent('ExitFinalized', exits)
  }
}
