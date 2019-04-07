import { ConfigManager, EthClient } from '../../../interfaces'
import { Process } from '../../common'
import { DefaultEthClient } from './eth-client'
import { CORE_CONFIG_KEYS } from '../constants'

export class DefaultEthClientProcess extends Process<EthClient> {
  constructor(private config: Process<ConfigManager>) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()
    const endpoint = this.config.subject.get(CORE_CONFIG_KEYS.ETHEREUM_ENDPOINT)
    this.subject = new DefaultEthClient(endpoint)
  }
}
