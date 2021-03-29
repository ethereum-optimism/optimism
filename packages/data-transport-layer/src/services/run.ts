/* Imports: External */
import * as dotenv from 'dotenv'
import Config from 'bcfg' // TODO: Add some types for bcfg if we get the chance.

/* Imports: Internal */
import { L1DataTransportService } from './main/service'

interface Bcfg {
  load: (options: { env?: boolean; argv?: boolean }) => void
  str: (name: string, defaultValue?: string) => string
  uint: (name: string, defaultValue?: number) => number
  bool: (name: string, defaultValue?: boolean) => boolean
}

;(async () => {
  try {
    dotenv.config()

    const config: Bcfg = new Config('data-transport-layer')
    config.load({
      env: true,
      argv: true,
    })

    const service = new L1DataTransportService({
      dbPath: config.str('dbPath', './db'),
      port: config.uint('serverPort', 7878),
      hostname: config.str('serverHostname', 'localhost'),
      confirmations: config.uint('confirmations', 35),
      l1RpcProvider: config.str('l1RpcEndpoint'),
      addressManager: config.str('addressManager'),
      pollingInterval: config.uint('pollingInterval', 5000),
      logsPerPollingInterval: config.uint('logsPerPollingInterval', 2000),
      dangerouslyCatchAllErrors: config.bool(
        'dangerouslyCatchAllErrors',
        false
      ),
      l2RpcProvider: config.str('l2RpcEndpoint'),
      l2ChainId: config.uint('l2ChainId'),
      syncFromL1: config.bool('syncFromL1', true),
      syncFromL2: config.bool('syncFromL2', false),
      showUnconfirmedTransactions: config.bool('syncFromL2', false),
      transactionsPerPollingInterval: config.uint(
        'transactionsPerPollingInterval',
        1000
      ),
      legacySequencerCompatibility: config.bool(
        'legacySequencerCompatibility',
        false
      ),
      stopL2SyncAtBlock: config.uint('stopL2SyncAtBlock'),
    })

    await service.start()
  } catch (err) {
    console.error(
      `Well, that's that. We ran into a fatal error. Here's the dump. Goodbye!`
    )

    throw err
  }
})()
