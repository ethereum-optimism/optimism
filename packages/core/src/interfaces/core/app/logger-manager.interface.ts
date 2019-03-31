import { Logger } from '../../common'

/**
 * LoggerManager is used to create and manage Logger instances.
 */
export interface LoggerManager {
  /**
   * Creates a logger instance
   * @param namespace to log to.
   * @returns the logger instance.
   */
  create(namespace: string): Logger
}
