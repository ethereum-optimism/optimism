import debug from 'debug'
import { Logger } from '../types'

export const getLogger = (
  identifier: string,
  isTest: boolean = false
): Logger => {
  const testString = isTest ? 'test:' : ''
  return {
    debug: debug(`${testString}debug:${identifier}`),
    info: debug(`${testString}info:${identifier}`),
    error: debug(`${testString}error:${identifier}`),
  }
}

export const logError = (logger: Logger, message: string, e: Error): void => {
  logger.error(`${message}. 
    Error: ${e.message}. 
    Stack: ${e.stack}`)
}
