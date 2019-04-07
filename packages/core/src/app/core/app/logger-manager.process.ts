import { LoggerManager } from '../../../interfaces'
import { Process } from '../../common'
import { DefaultLoggerManager } from './logger-manager'

export class DefaultLoggerManagerProcess extends Process<LoggerManager> {
  protected async onStart(): Promise<void> {
    this.subject = new DefaultLoggerManager()
  }
}
