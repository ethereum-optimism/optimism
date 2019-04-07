import { AbstractLevelDOWNConstructor } from 'abstract-leveldown'

import {
  ConfigManager,
  LoggerManager,
  DBManager,
  EthClient,
  KeyManager,
} from '../../interfaces'
import { Process, BaseApp } from '../common'
import { DefaultConfigManagerProcess, DefaultLoggerManagerProcess } from './app'
import { DefaultDBManagerProxy } from './db'
import { DefaultEthClientProcess, DefaultKeyManagerProcess } from './eth'

export interface CoreAppConfig {
  ETHEREUM_ENDPOINT: string
  BASE_DB_PATH: string
  DB_BACKEND: AbstractLevelDOWNConstructor
}

export class CoreApp extends BaseApp {
  public readonly configManager: Process<ConfigManager>
  public readonly loggerManager: Process<LoggerManager>
  public readonly dbManager: Process<DBManager>
  public readonly ethClient: Process<EthClient>
  public readonly keyManager: Process<KeyManager>

  constructor(config: CoreAppConfig) {
    super()

    this.configManager = new DefaultConfigManagerProcess(config)
    this.loggerManager = new DefaultLoggerManagerProcess()
    this.dbManager = new DefaultDBManagerProxy(this.configManager)
    this.ethClient = new DefaultEthClientProcess(this.configManager)
    this.keyManager = new DefaultKeyManagerProcess(this.dbManager)

    this.register('ConfigManager', this.configManager)
    this.register('LogCollector', this.loggerManager)
    this.register('DBManager', this.dbManager)
    this.register('EthClient', this.ethClient)
    this.register('KeyManager', this.keyManager)
  }
}
