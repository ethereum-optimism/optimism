import { Logger } from '../../common'

/**
 * LoggerManager is used to create and manage Logger instances.
 */
export interface LoggerManager {
  create(namespace: string): Logger
}
