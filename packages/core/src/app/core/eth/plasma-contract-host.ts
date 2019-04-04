import { EthClient, MessageBus } from '../../../interfaces'
import { BaseRunnable } from '../../common'
import { PlasmaContractWrapper } from './plasma-contract-wrapper'

export class PlasmaContractHost extends BaseRunnable {
  private _contract: PlasmaContractWrapper

  constructor(private ethClient: EthClient, private messageBus: MessageBus) {
    super()
  }

  get contract(): PlasmaContractWrapper {
    return this._contract
  }

  public async onStart(): Promise<void> {
    this.messageBus.on('contract:address', this.onAddressFound.bind(this))
  }

  public async onStop(): Promise<void> {
    this.messageBus.off('contract:address', this.onAddressFound.bind(this))
  }

  private onAddressFound(address: string): void {
    this._contract = new PlasmaContractWrapper(this.ethClient.web3, address)
  }
}
