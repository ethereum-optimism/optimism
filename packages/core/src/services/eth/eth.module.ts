/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import { EthService } from './eth.service'
import { ContractService } from './contract.service'

@Module({
  services: [EthService, ContractService],
})
export class EthModule {}
