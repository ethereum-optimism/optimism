import { Provider } from '@ethersproject/abstract-provider'
import { sleep } from '@eth-optimism/core-utils'

import { Logger } from './logger'

/**
 * Waits for an Ethers provider to be connected.
 *
 * @param provider Ethers provider to check.
 * @param opts Options for the function.
 * @param opts.logger Logger to use.
 * @param opts.intervalMs Interval to wait between checks.
 * @param opts.name Name of the provider for logs.
 */
export const waitForProvider = async (
  provider: Provider,
  opts?: {
    logger?: Logger
    intervalMs?: number
    name?: string
  }
) => {
  const name = opts?.name || 'target'
  opts?.logger?.info(`waiting for ${name} provider...`)
  let connected = false
  while (!connected) {
    try {
      await provider.getBlockNumber()
      connected = true
    } catch (e) {
      opts?.logger?.info(`${name} provider not connected, retrying...`)
      await sleep(opts?.intervalMs || 15000)
    }
  }
  opts?.logger?.info(`${name} provider connected`)
}
