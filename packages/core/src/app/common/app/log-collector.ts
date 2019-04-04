import { MessageBus } from '../../../interfaces'
import { BaseRunnable } from '../app'
import { DebugLogger } from '../utils'
import { DefaultMessageBus } from './message-bus'
import { Service } from '@nestd/core';

interface LogMessage {
  type: 'log' | 'warn' | 'error'
  namespace: string
  message: string
  trace: any
}

/**
 * Simple log collector that joins logs from the message bus.
 */
@Service()
export class DefaultLogCollector extends BaseRunnable {
  private loggers: Record<string, DebugLogger>

  constructor(private messageBus: DefaultMessageBus) {
    super()
  }

  public async onStart(): Promise<void> {
    this.messageBus.on('log', this.onLog.bind(this))
  }

  public async onStop(): Promise<void> {
    this.messageBus.off('log', this.onLog.bind(this))
  }

  /**
   * Handles a new log message.
   * @param log Log message to handle.
   */
  private onLog(log: LogMessage): void {
    const logger = this.getLogger(log.namespace)
    logger[log.type](log.message, log.trace)
  }

  /**
   * Creates a new logger using `debug`.
   * @param namespace Namespace to log under.
   * @returns the logger instance.
   */
  private getLogger(namespace: string): DebugLogger {
    if (namespace in this.loggers) {
      return this.loggers[namespace]
    }

    const logger = new DebugLogger(namespace)
    this.loggers[namespace] = logger
    return logger
  }
}
