import { AbstractLevelDOWNConstructor } from 'abstract-leveldown'

import {
  ConfigManager,
  LoggerManager,
  DBManager,
  EthClient,
  KeyManager,
  RpcServer,
} from '../../interfaces'
import { Process, BaseApp } from '../common'
import { SimpleConfigManagerProcess, DebugLoggerManagerProcess } from './app'
import { SimpleDBManagerProcess } from './db'
import { Web3EthClientProcess, SimpleKeyManagerProcess } from './eth'
import { SimpleJsonRpcServerProcess } from './net'

export interface CoreAppConfig {
  ETHEREUM_ENDPOINT: string
  BASE_DB_PATH: string
  DB_BACKEND: AbstractLevelDOWNConstructor
  RPC_SERVER_PORT: number
  RPC_SERVER_HOSTNAME: string
}

/**
 * Core L2 app. Contains all of the basic components
 * necessary for a layer 2 system.
 */
export class CoreApp extends BaseApp {
  public readonly configManager: Process<ConfigManager>
  public readonly loggerManager: Process<LoggerManager>
  public readonly dbManager: Process<DBManager>
  public readonly ethClient: Process<EthClient>
  public readonly keyManager: Process<KeyManager>
  public readonly rpcServer: Process<RpcServer>

  /**
   * Creates the app.
   * @param config Configuration for the app.
   */
  constructor(config: CoreAppConfig) {
    super()

    this.configManager = new SimpleConfigManagerProcess(config)
    this.loggerManager = new DebugLoggerManagerProcess()
    this.dbManager = new SimpleDBManagerProcess(this.configManager)
    this.ethClient = new Web3EthClientProcess(this.configManager)
    this.keyManager = new SimpleKeyManagerProcess(this.dbManager)
    this.rpcServer = new SimpleJsonRpcServerProcess(this.configManager)

    this.register('ConfigManager', this.configManager)
    this.register('LogCollector', this.loggerManager)
    this.register('DBManager', this.dbManager)
    this.register('EthClient', this.ethClient)
    this.register('KeyManager', this.keyManager)
  }
}
