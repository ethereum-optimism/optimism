import { vi } from 'vitest'

/**
 * A test util for watching console output
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
