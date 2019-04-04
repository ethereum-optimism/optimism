import { BaseRunnable } from '../../common'
import { EthClient, ConfigManager, MessageBus } from '../../../interfaces'
import { RegistryContractWrapper } from './registry-contract-wrapper'

/**
 * Responsible for determining the plasma chain contract's
 * address and putting it on the message bus.
 */
export class PlasmaContractDetector extends BaseRunnable {
  constructor(
    private ethClient: EthClient,
    private config: ConfigManager,
    private messageBus: MessageBus
  ) {
    super()
  }

  public async onStart(): Promise<void> {
    const registryAddress = this.config.get('REGISTRY_ADDRESS')
    const registry = new RegistryContractWrapper(
      this.ethClient.web3,
      registryAddress
    )

    const plasmaChainName = this.config.get('PLASMA_CHAIN_NAME')
    const plasmaChainAddress = await registry.getPlasmaChainAddress(
      plasmaChainName
    )

    this.messageBus.emit('contract:address', plasmaChainAddress)
  }
}
