/* External Imports */
import {BaseDB, DB, getLevelInstance, newInMemoryDB} from '@eth-optimism/core-db'
import {getLogger} from '@eth-optimism/core-utils'

import * as rimraf from 'rimraf'

/* Internal Imports */
import * as fs from "fs"
import {Environment} from './environment'

const log = getLogger('filepath-util')

/**
 * Initializes filesystem DB paths. This will also purge all data if the `CLEAR_DATA_KEY` has changed.
 */
export const initializeDBPaths = (dbPath: string, isTestMode: boolean) => {
  if (isTestMode) {
    return
  }

  if (!fs.existsSync(dbPath)) {
    makeDataDirectory(dbPath)
  } else {
    if (Environment.clearDataKey() && !fs.existsSync(getClearDataFilePath(dbPath))) {
      log.info(`Detected change in CLEAR_DATA_KEY. Purging data...`)
      rimraf.sync(`${dbPath}/{*,.*}`)
      log.info(
        `Data purged from '${dbPath}/{*,.*}'`
      )
      if (Environment.localL1NodePersistentDbPath()) {
        rimraf.sync(`${Environment.localL1NodePersistentDbPath()}/{*,.*}`)
        log.info(
          `Local L1 node data purged from '${Environment.localL1NodePersistentDbPath()}/{*,.*}'`
        )
      }
      if (Environment.localL2NodePersistentDbPath()) {
        rimraf.sync(`${Environment.localL2NodePersistentDbPath()}/{*,.*}`)
        log.info(
          `Local L2 node data purged from '${Environment.localL2NodePersistentDbPath()}/{*,.*}'`
        )
      }
      makeDataDirectory()
    }
  }
}

/**
 * Makes the data directory for this full node and adds a clear data key file if it is configured to use one.
 */
export const makeDataDirectory = () => {
  fs.mkdirSync(Environment.l2RpcServerPersistentDbPath(), { recursive: true })
  if (Environment.clearDataKey()) {
    fs.writeFileSync(getClearDataFilePath(), '')
  }
}

/**
 * Gets the filepath of the "Clear Data" file that dictates whether or not all filesystem data should be cleared on startup.
 */
export const getClearDataFilePath = () => {
  return `${Environment.l2RpcServerPersistentDbPath()}/.clear_data_key_${Environment.clearDataKey()}`
}
