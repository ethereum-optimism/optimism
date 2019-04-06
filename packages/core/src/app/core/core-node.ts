import {
  ConfigManager,
  LogCollector,
  MessageBus,
  DBManager,
  EthClient,
  EventWatcher,
  KeyManager,
} from '../../interfaces'
import {
  DefaultConfigManager,
  DefaultMessageBus,
  DefaultLogCollector,
} from './app'
import { DefaultDBManager } from './db'
import { DefaultEthClient, DefaultEventWatcher, DefaultKeyManager } from './eth'

interface Runnable {
  start(): Promise<void>
  stop(): Promise<void>
}

const isRunnable = (process: any): process is Runnable => {
  return process.start !== undefined && process.stop !== undefined
}

export class Node {
  private processes: Record<string, any>

  public register(name: string, process: any): void {
    if (name in this.processes) {
      throw new Error(`Process already registered: ${name}`)
    }

    this.processes[name] = process
  }

  public async start(): Promise<void> {
    const runnables = this.getRunnables()
    await Promise.all(
      runnables.map((runnable) => {
        return runnable.start()
      })
    )
  }

  public async stop(): Promise<void> {
    const runnables = this.getRunnables()
    await Promise.all(
      runnables.map((runnable) => {
        return runnable.stop()
      })
    )
  }

  private getRunnables(): Runnable[] {
    const runnables = []
    for (const name of Object.keys(this.processes)) {
      const process = this.processes[name]
      if (isRunnable(process)) {
        runnables.push(process)
      }
    }
    return runnables
  }
}

export class CoreNode extends Node {
  public readonly configManager: ConfigManager
  public readonly messageBus: MessageBus
  public readonly logCollector: LogCollector
  public readonly dbManager: DBManager
  public readonly ethClient: EthClient
  public readonly eventWatcher: EventWatcher
  public readonly keyManager: KeyManager

  constructor() {
    super()

    this.configManager = new DefaultConfigManager()
    this.messageBus = new DefaultMessageBus()
    this.logCollector = new DefaultLogCollector(this.messageBus)
    this.dbManager = new DefaultDBManager(
      this.configManager.get('BASE_DB_PATH'),
      this.configManager.get('DB_BACKEND')
    )
    this.ethClient = new DefaultEthClient(
      this.configManager.get('ETHEREUM_ENDPOINT')
    )
    this.eventWatcher = new DefaultEventWatcher(this.messageBus, this.ethClient)
    this.keyManager = new DefaultKeyManager()

    this.register('ConfigManager', this.configManager)
    this.register('MessageBus', this.messageBus)
    this.register('LogCollector', this.logCollector)
    this.register('DBManager', this.dbManager)
    this.register('EthClient', this.ethClient)
    this.register('EventWatcher', this.eventWatcher)
    this.register('KeyManager', this.keyManager)
  }
}
