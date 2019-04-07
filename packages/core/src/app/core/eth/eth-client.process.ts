import { ConfigManager, EthClient } from '../../../interfaces'
import { Process } from '../../common'
import { DefaultEthClient } from './eth-client'

export class DefaultEthClientProcess extends Process<EthClient> {
  constructor(private config: Process<ConfigManager>) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()
    const endpoint = this.config.subject.get('ETHEREUM_ENDPOINT')
    this.subject = new DefaultEthClient(endpoint)
  }
}
