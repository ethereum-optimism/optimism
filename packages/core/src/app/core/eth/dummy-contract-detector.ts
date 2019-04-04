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
export class DummyContractDetector extends BaseRunnable {
  constructor(
    private ethClient: DefaultEthClient,
    private config: DefaultConfigManager,
    private messageBus: DefaultMessageBus
  ) {
    super()
  }

  public async onStart(): Promise<void> {
    setTimeout(() => {
      this.messageBus.emit('contract:address', '0x0000000000000000000000000000000000000000')
    }, 50)
  }
}
