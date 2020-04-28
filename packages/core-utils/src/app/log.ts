import debug from 'debug'
import { Logger } from '../types'

export const LOG_NEWLINE_STRING = '<\n>'
export const joinNewlinesAndDebug = (logs :string) =>
  debug(logs.replace("\n", LOG_NEWLINE_STRING))

export const getLogger = (
  identifier: string,
  isTest: boolean = false
): Logger => {
  const testString = isTest ? 'test:' : ''
  return {
    debug: joinNewlinesAndDebug(`${testString}debug:${identifier}`),
    info: joinNewlinesAndDebug(`${testString}info:${identifier}`),
    warn: joinNewlinesAndDebug(`${testString}warn:${identifier}`),
    error: joinNewlinesAndDebug(`${testString}error:${identifier}`),
  }
}

export const logError = (logger: Logger, message: string, e: Error): void => {
  logger.error(`${message}. 
    Error: ${e.message}. 
    Stack: ${e.stack}`)
}
