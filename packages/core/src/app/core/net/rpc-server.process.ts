/* Internal Imports */
import { ConfigManager, RpcServer } from '../../../interfaces'
import { Process } from '../../common'
import { CORE_CONFIG_KEYS } from '../constants'
import { SimpleJsonRpcServer } from './rpc-server'

/**
 * Creates a JSON-RPC server.
 */
export class SimpleJsonRpcServerProcess extends Process<RpcServer> {
  /**
   * Creates the process.
   * @param config Process used to load config values.
   */
  constructor(private config: Process<ConfigManager>) {
    super()
  }

  /**
   * Creates the server.
   * Waits for config to be ready before
   * creating the server and listening.
   */
  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()

    const port = this.config.subject.get(CORE_CONFIG_KEYS.RPC_SERVER_PORT)
    const hostname = this.config.subject.get(
      CORE_CONFIG_KEYS.RPC_SERVER_HOSTNAME
    )
    this.subject = new SimpleJsonRpcServer({}, port, hostname)
    await this.subject.listen()
  }
}
