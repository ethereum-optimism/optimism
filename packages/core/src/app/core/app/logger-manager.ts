import { LoggerManager } from '../../../interfaces'
import { DebugLogger } from '../../common'

export class DefaultLoggerManager implements LoggerManager {
  private loggers: Record<string, DebugLogger>

  public create(namespace: string): DebugLogger {
    if (namespace in this.loggers) {
      return this.loggers[namespace]
    }

    const logger = new DebugLogger(namespace)
    this.loggers[namespace] = logger
    return logger
  }
}
