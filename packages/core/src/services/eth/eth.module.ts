/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import { EthDataService } from './eth-data.service'
import { ContractService } from './contract.service'
import { WalletService } from './wallet.service'

@Module({
  services: [EthDataService, ContractService, WalletService],
})
export class EthModule {}
