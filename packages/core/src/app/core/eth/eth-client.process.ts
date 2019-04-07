import { ConfigManager, EthClient } from '../../../interfaces'
import { Process } from '../../common'
import { CORE_CONFIG_KEYS } from '../constants'
import { Web3EthClient } from './eth-client'

/**
 * Process that initializes an EthClient instance.
 */
export class Web3EthClientProcess extends Process<EthClient> {
  /**
   * Creates the process.
   * @param config Config process used to load config values.
   */
  constructor(private config: Process<ConfigManager>) {
    super()
  }

  /**
   * Creates the EthClient instance.
   * Waits for config to be available.
   */
  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()

    const endpoint = this.config.subject.get(CORE_CONFIG_KEYS.ETHEREUM_ENDPOINT)
    this.subject = new Web3EthClient(endpoint)
  }
}
