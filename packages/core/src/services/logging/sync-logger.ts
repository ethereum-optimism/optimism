import { LoggerService } from './logger.service'

/**
 * Wrapper that streams log messages to
 * the logging service.
 */
export class SyncLogger {
  constructor(
    private readonly namespace: string,
    private readonly logger: LoggerService
  ) {}

  /**
   * Logs a message.
   * @param message Message to log.
   */
  public log(message: string): void {
    this.logger.log(this.namespace, message)
  }

  /**
   * Logs an error.
   * @param message Error message to log.
   * @param [trace] An optional error trace.
   */
  public error(message: string, trace?: any): void {
    this.logger.error(this.namespace, message, trace)
  }

  /**
   * Logs a warning.
   * @param message Warning message to log.
   */
  public warn(message: string): void {
    this.logger.warn(this.namespace, message)
  }
}
