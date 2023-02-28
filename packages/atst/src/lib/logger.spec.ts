import { afterEach, describe, expect, it, vi } from 'vitest'

import * as logger from './logger'
import { watchConsole } from '../test/watchConsole'

describe('logger', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe.each([
    { level: 'success' },
    { level: 'info' },
    { level: 'log' },
    { level: 'warn' },
    { level: 'error' },
    // eslint-disable-next-line no-template-curly-in-string
  ])('${level}()', ({ level }) => {
    it(`logs message "${level}"`, () => {
      const spy = vi.spyOn(logger, level as 'info')
      const consoleUtil = watchConsole()
      const loggerFn = logger[level]
      loggerFn(level)
      expect(spy).toHaveBeenCalledWith(level)
      expect(consoleUtil.formatted).toMatchSnapshot()
    })
  })
})
