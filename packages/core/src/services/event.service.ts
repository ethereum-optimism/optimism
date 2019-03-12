/* External Imports */
import { EventEmitter } from 'events'
import { Service } from '@nestd/core'

@Service()
export class EventService extends EventEmitter {
  public event(namespace: string, event: string, ...args: any[]): void {
    this.emit(`${namespace}.${event}`, args)
  }
}
