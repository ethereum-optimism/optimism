/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import { EthDataService } from './eth-data.service'
import { ContractService } from './contract.service'
import { WalletService } from './wallet.service'
import { EthEventWatcherService } from './events/eth-event-watcher.service'
import { EthEventHandlerService } from './events/eth-event-handler.service'

@Module({
  services: [
    EthDataService,
    ContractService,
    WalletService,
    EthEventWatcherService,
    EthEventHandlerService,
  ],
})
export class EthModule {}
