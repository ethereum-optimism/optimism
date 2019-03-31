import { LoggerManager } from '../../../interfaces'
import { DebugLogger } from '../../common'

/**
 * Manages and creates new logger instances.
 * Uses `debug` by default.
 */
export class DefaultLoggerManager implements LoggerManager {
  private loggers: Record<string, DebugLogger>

  /**
   * Creates a new logger using `debug`.
   * @param namespace Namespace to log under.
   * @returns the logger instance.
   */
  public create(namespace: string): DebugLogger {
    if (namespace in this.loggers) {
      return this.loggers[namespace]
    }

    const logger = new DebugLogger(namespace)
    this.loggers[namespace] = logger
    return logger
  }
}
