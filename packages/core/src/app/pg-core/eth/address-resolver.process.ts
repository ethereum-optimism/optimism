import { Process } from '../../common'
import { EthClient, ConfigManager, AddressResolver } from '../../../interfaces'
import { RegistryContractWrapper } from './registry-contract-wrapper'

export class DefaultAddressResolverProcess extends Process<AddressResolver> {
  constructor(
    private config: Process<ConfigManager>,
    private ethClient: Process<EthClient>
  ) {
    super()
  }

  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()
    await this.ethClient.waitUntilStarted()

    const registryAddress = this.config.subject.get('REGISTRY_ADDRESS')
    const registry = new RegistryContractWrapper(
      this.ethClient.subject.web3,
      registryAddress
    )

    const plasmaChainName = this.config.subject.get('PLASMA_CHAIN_NAME')
    const plasmaChainAddress = await registry.getPlasmaChainAddress(
      plasmaChainName
    )

    this.subject = {
      address: plasmaChainAddress,
    }
  }
}
