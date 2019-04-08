/* Internal Imports */
import { EthClient, ConfigManager, AddressResolver } from '../../../interfaces'
import { Process } from '../../common'
import { PG_CORE_CONFIG_KEYS } from '../constants'
import { RegistryContractWrapper } from './registry-contract-wrapper'

/**
 * Process that resolves the address of the plasma chain.
 */
export class PGAddressResolverProcess extends Process<AddressResolver> {
  /**
   * Creates the process.
   * @param config Process used to load config values.
   * @param ethClient Process used to connect to Ethereum.
   */
  constructor(
    private config: Process<ConfigManager>,
    private ethClient: Process<EthClient>
  ) {
    super()
  }

  /**
   * Resolves the plasma chain address.
   * Waits for config and eth client to be ready
   * before querying the registry and resolving
   * the address.
   */
  protected async onStart(): Promise<void> {
    await this.config.waitUntilStarted()
    await this.ethClient.waitUntilStarted()

    // Connect to the registry.
    const registryAddress = this.config.subject.get(
      PG_CORE_CONFIG_KEYS.REGISTRY_ADDRESS
    )
    const registry = new RegistryContractWrapper(
      this.ethClient.subject.web3,
      registryAddress
    )

    // Get the plasma chain name.
    const plasmaChainName = this.config.subject.get(
      PG_CORE_CONFIG_KEYS.PLASMA_CHAIN_NAME
    )
    const plasmaChainAddress = await registry.getPlasmaChainAddress(
      plasmaChainName
    )

    this.subject = {
      address: plasmaChainAddress,
    }
  }
}
