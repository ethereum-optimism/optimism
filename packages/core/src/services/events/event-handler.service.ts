/* External Imports */
import { Service, OnStart } from '@nestd/core'

/* Services */
import { EventWatcherService } from './event-watcher.service'
import { EventService } from '../event.service'
import { LoggerService } from '../logger.service'

/* Internal Imports */
import { EthereumEvent } from '../../models/eth'
import {
  BlockSubmittedEvent,
  DepositEvent,
  ExitFinalizedEvent,
  ExitStartedEvent,
} from '../../models/events'

@Service()
export class EventHandlerService implements OnStart {
  private readonly name = 'eventHandler'

  constructor(
    private readonly logger: LoggerService,
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
    const deposits = events.map(DepositEvent.from)
    deposits.forEach((deposit) => {
      this.logger.log(
        this.name,
        `Detected new deposit of ${deposit.amount} [${deposit.token}] for ${
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
    const blocks = events.map(BlockSubmittedEvent.from)
    blocks.forEach((block) => {
      this.logger.log(
        this.name,
        `Detected block #${block.number}: ${block.hash}`
      )
    })
    this.emitContractEvent('BlockSubmitted', blocks)
  }

  /**
   * Handles ExitStarted events.
   * @param events ExitStarted events.
   */
  private onExitStarted(events: EthereumEvent[]): void {
    const exits = events.map(ExitStartedEvent.from)
    exits.forEach((exit) => {
      this.logger.log(this.name, `Detected new started exit: ${exit.id}`)
    })
    this.emitContractEvent('ExitStarted', exits)
  }

  /**
   * Handles ExitFinalized events.
   * @param events ExitFinalized events.
   */
  private onExitFinalized(events: EthereumEvent[]): void {
    const exits = events.map(ExitFinalizedEvent.from)
    exits.forEach((exit) => {
      this.logger.log(this.name, `Detected new finalized exit: ${exit.id}`)
    })
    this.emitContractEvent('ExitFinalized', exits)
  }
}
