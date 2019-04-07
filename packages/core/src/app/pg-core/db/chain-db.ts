import { KeyValueStore, ChainDB } from '../../../interfaces'
import { BaseKey } from '../../common'

/**
 * Basic ChainDB implementation that provides a
 * nice interface to the chain database.
 */
export class PGChainDB implements ChainDB {
  private statedb: KeyValueStore
  private historydb: KeyValueStore

  /**
   * Creates the wrapper.
   * @param db Database to interact with.
   */
  constructor(private db: KeyValueStore) {
    const statePrefix = new BaseKey('s')
    this.statedb = db.bucket(statePrefix.encode())

    const historyPrefix = new BaseKey('h')
    this.historydb = db.bucket(historyPrefix.encode())
  }
}
