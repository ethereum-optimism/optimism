import { afterEach, describe, expect, it, vi } from 'vitest'

import * as logger from './logger'

const mockLog = vi.fn()

/**
 * We will use this test util later to write tests for the cli
 */
export const watchConsole = () => {
  type Console = 'info' | 'log' | 'warn' | 'error'
  const output: { [_ in Console | 'all']: string[] } = {
    info: [],
    log: [],
    warn: [],
    error: [],
    all: [],
  }
  const handleOutput = (method: Console) => {
    return (message: string) => {
      output[method].push(message)
      output.all.push(message)
    }
  }
  return {
    debug: console.debug,
    info: vi.spyOn(console, 'info').mockImplementation(handleOutput('info')),
    log: vi.spyOn(console, 'log').mockImplementation(handleOutput('log')),
    warn: vi.spyOn(console, 'warn').mockImplementation(handleOutput('warn')),
    error: vi.spyOn(console, 'error').mockImplementation(handleOutput('error')),
    output,
    get formatted() {
      return output.all.join('\n')
    },
  }
}

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
      const spy = vi.spyOn(logger, level as any)
      spy.mockImplementation(mockLog)
      const loggerFn = (logger as any)[level]
      loggerFn(level)
      expect(spy).toHaveBeenCalledWith(level)
    })
  })

  it('spinner', () => {
    const console = watchConsole()
    const spinner = logger.spinner()

    spinner.start('Foo bar baz')
    spinner.succeed('Foo bar baz')
    spinner.fail('Foo bar baz')
    expect(console.formatted).toMatchInlineSnapshot('""')
  })
})
