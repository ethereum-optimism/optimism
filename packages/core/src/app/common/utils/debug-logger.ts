import debug, { Debugger } from 'debug'

import { Logger } from '../../../interfaces'

declare const process: any

export class DebugLogger implements Logger {
  private logger: Debugger

  constructor(readonly namespace: string) {
    this.logger = debug(namespace)
    debug.enable(namespace)
  }

  /**
   * Logs a message.
   * @param message to log.
   */
  public log(message: string): void {
    this.logger(
      `[pg] ${process.pid}   -   ${new Date(
        Date.now()
      ).toLocaleString()}   -   ${message}`
    )
  }

  /**
   * Logs an error.
   * @param message to log.
   * @param [trace] for the error.
   */
  public error(message: string, trace?: string): void {
    this.log(`ERROR: ${message}\n${trace}`)
  }

  /**
   * Logs a warning.
   * @param message to log.
   */
  public warn(message: string): void {
    this.log(`WARNING: ${message}`)
  }
}
