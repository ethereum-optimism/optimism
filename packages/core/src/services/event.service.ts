/* External Imports */
import { EventEmitter } from 'events'
import { Service } from '@nestd/core'

/**
 * Service used for relaying events between
 * different services.
 */
@Service()
export class EventService extends EventEmitter {
  /**
   * Emits an event for a given namespace.
   * @param namespace Namespace to emit an event for.
   * @param event: Name of the event to emit.
   * @param args: Any additional event arguments.
   */
  public event(namespace: string, event: string, ...args: any[]): void {
    this.emit(`${namespace}.${event}`, args)
  }
}
