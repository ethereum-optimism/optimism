import debug from 'debug'
import { Logger } from '../../types/utils'

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
