import debug, { Debugger } from 'debug'

import { Logger } from '../../../interfaces'

export class DebugLogger implements Logger {
  private logger: Debugger

  constructor(readonly namespace: string) {
    this.logger = debug(namespace)
  }

  /**
   * Logs a message.
   * @param message to log.
   */
  log(message: string): void {
    this.logger(message)
  }

  /**
   * Logs an error.
   * @param message to log.
   * @param [trace] for the error.
   */
  error(message: string, trace?: string): void {
    this.log(`ERROR: ${message}\n${trace}`)
  }

  /**
   * Logs a warning.
   * @param message to log.
   */
  warn(message: string): void {
    this.log(`WARNING: ${message}`)
  }
}
