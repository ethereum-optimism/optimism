import { Process } from '../common'
import { CoreApp, CoreAppConfig } from '../core'
import {
  AddressResolver,
  HistoryManager,
  StateManager,
  ChainDB,
} from '../../interfaces'
import { PGAddressResolverProcess } from './eth'
import { PGChainDBProcess } from './db'
import { PGStateManagerProcess, PGHistoryManagerProcoess } from './state'

export interface PGCoreAppConfig extends CoreAppConfig {
  PLASMA_CHAIN_NAME: string
  REGISTRY_ADDRESS: string
}

export class PGCoreApp extends CoreApp {
  protected addressResolver: Process<AddressResolver>
  protected chaindb: Process<ChainDB>
  protected historyManager: Process<HistoryManager>
  protected stateManager: Process<StateManager>

  constructor(config: PGCoreAppConfig) {
    super(config)

    this.addressResolver = new PGAddressResolverProcess(
      this.configManager,
      this.ethClient
    )
    this.chaindb = new PGChainDBProcess(this.addressResolver, this.dbManager)
    this.historyManager = new PGHistoryManagerProcoess(this.chaindb)
    this.stateManager = new PGStateManagerProcess(this.chaindb)

    this.register('AddressResolver', this.addressResolver)
    this.register('ChainDB', this.chaindb)
    this.register('HistoryManager', this.historyManager)
    this.register('StateManager', this.stateManager)
  }
}
