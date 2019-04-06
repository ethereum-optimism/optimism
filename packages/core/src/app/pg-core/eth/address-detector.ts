import { Process } from '../../common'
import { EthClient, ConfigManager, MessageBus } from '../../../interfaces'
import { RegistryContractWrapper } from './registry-contract-wrapper'

export class AddressDetector extends Process {
  constructor(
    private messageBus: MessageBus,
    private ethClient: EthClient,
    private config: ConfigManager
  ) {
    super()
  }

  protected async onStart(): Promise<void> {
    const registryAddress = this.config.get('REGISTRY_ADDRESS')
    const registry = new RegistryContractWrapper(
      this.ethClient.web3,
      registryAddress
    )

    const plasmaChainName = this.config.get('PLASMA_CHAIN_NAME')
    const plasmaChainAddress = await registry.getPlasmaChainAddress(
      plasmaChainName
    )

    this.messageBus.emit('ADDRESS_FOUND', plasmaChainAddress)
  }
}
