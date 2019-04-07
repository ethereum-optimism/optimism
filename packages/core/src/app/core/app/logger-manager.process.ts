import { LoggerManager } from '../../../interfaces'
import { Process } from '../../common'
import { DebugLoggerManager } from './logger-manager'

/**
 * Simple wrapper process for the debug logger manager.
 */
export class DebugLoggerManagerProcess extends Process<LoggerManager> {
  /**
   * Creates the instance of the logger manager.
   */
  protected async onStart(): Promise<void> {
    this.subject = new DebugLoggerManager()
  }
}
