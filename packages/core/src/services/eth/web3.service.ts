/* External Imports */
import { Service, OnStart } from '@nestd/core'
import Web3 from 'web3'

/* Services */
import { ConfigService } from '../config.service'

/* Internal Imports */
import { CONFIG } from '../../constants'

@Service()
export class Web3Service implements OnStart {
  private _web3: Web3

  constructor(private readonly config: ConfigService) {}

  public async onStart(): Promise<void> {
    this._web3 = new Web3(
      new Web3.providers.HttpProvider(this.ethereumEndpoint())
    )
  }

  /**
   * @returns the current Web3 instance.
   */
  get web3(): Web3 {
    return this._web3
  }

  /**
   * @returns `true` if the node is connected to Ethereum, `false` otherwise.
   */
  public async connected(): Promise<boolean> {
    if (!this.web3) {
      return false
    }

    try {
      await this.web3.eth.net.isListening()
      return true
    } catch (e) {
      return false
    }
  }

  /**
   * @returns the current Ethereum endpoint.
   */
  private ethereumEndpoint(): string {
    return this.config.get(CONFIG.ETHEREUM_ENDPOINT)
  }
}
