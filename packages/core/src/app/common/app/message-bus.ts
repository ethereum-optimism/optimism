import { EventEmitter } from 'events'

import { MessageBus } from '../../../interfaces'
import { BaseRunnable } from '..'

/**
 * Simple message bus that uses Node.js event emitters.
 */
export class DefaultMessageBus extends BaseRunnable implements MessageBus {
  private emitter = new EventEmitter()

  public async onStop(): Promise<void> {
    this.emitter.removeAllListeners()
  }

  /**
   * Emits an event to all listeners.
   * @param event Event to emit.
   * @param args Arguments to the event.
   */
  public emit(event: string, ...args: any[]): void {
    this.emitter.emit(event, args)
  }

  /**
   * Listens for an event.
   * @param event Event to listen for.
   * @param listener Function to call when the event is triggered.
   */
  public on(event: string, listener: (...args: any[]) => any): void {
    this.emitter.on(event, listener)
  }

  /**
   * Stops listening for an event.
   * @param event Event to stop listening for.
   * @param listener Function that was used to listen.
   */
  public off(event: string, listener: (...args: any[]) => any): void {
    this.emitter.off(event, listener)
  }
}
