import debug from 'debug'
import { Logger } from '../types'

export const LOG_NEWLINE_STRING = ' <\\n> '

/**
 * Gets a logger specific to the provided identifier.
 *
 * @param identifier The identifier to use to tag log statements from this logger.
 * @param isTest Whether or not this is a test logger.
 * @param debugToUseTestOnly The debug instance to use *should only be used for tests*
 * @returns a Logger instance.
 */
export const getLogger = (
  identifier: string,
  isTest: boolean = false,
  debugToUseTestOnly?: debug
): Logger => {
  const testString = isTest ? 'test:' : ''
  return {
    debug: getLogFunction(
      `${testString}debug:${identifier}`,
      debugToUseTestOnly
    ),
    info: getLogFunction(`${testString}info:${identifier}`, debugToUseTestOnly),
    warn: getLogFunction(`${testString}warn:${identifier}`, debugToUseTestOnly),
    error: getLogFunction(
      `${testString}error:${identifier}`,
      debugToUseTestOnly
    ),
  }
}

export const logError = (logger: Logger, message: string, e: Error): void => {
  logger.error(`${message}. 
    Error: ${e.message}. 
    Stack: ${e.stack}`)
}

/**
 * Converts one or more items to log into a single line string.
 *
 * @param logs The array of items to log
 * @returns The single-line string.
 */
const joinNewLines = (...logs: any[]): string => {
  const stringifiedLogs = []
  for (const l of logs) {
    if (typeof l !== 'string') {
      stringifiedLogs.push(JSON.stringify(l))
    } else {
      stringifiedLogs.push(l)
    }
  }

  return stringifiedLogs.join(' ').replace(/\n/g, LOG_NEWLINE_STRING)
}

/**
 * Creates a debug instance with the provided identifier and wraps
 * its only function in a function that makes the strings to be logged a single
 * line before calling debug(identifier)(log).
 *
 * @param identifier The identifier used to prepend this log
 * @param debugToUseTestOnly The debug instance to use *should only be used for tests*
 * @returns The log function for the provided identifier
 */
const getLogFunction = (
  identifier: string,
  debugToUseTestOnly: debug = debug
): any => {
  const d = debugToUseTestOnly(identifier)
  return (...logs: any[]): any => {
    const singleLine = joinNewLines(...logs)
    return d(singleLine)
  }
}
