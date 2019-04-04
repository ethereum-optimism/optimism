import { BaseRunnable, DefaultEthClient } from '../../common'
import { EthClient, ConfigManager, MessageBus } from '../../../interfaces'
import { RegistryContractWrapper } from './registry-contract-wrapper'
import { DefaultConfigManager } from '../../common/app/config-manager';
import { DefaultMessageBus } from '../../common/app/message-bus';
import { Service } from '@nestd/core';

/**
 * Responsible for determining the plasma chain contract's
 * address and putting it on the message bus.
 */
@Service()
export class PlasmaContractDetector extends BaseRunnable {
  constructor(
    private ethClient: DefaultEthClient,
    private config: DefaultConfigManager,
    private messageBus: DefaultMessageBus
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
