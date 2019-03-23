/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import {
  SyncService,
  GuardService,
  WalletService,
  JsonRpcService,
  OperatorService,
  ChainService,
  EthModule,
  DBModule,
  ProofModule,
} from './services'

@Module({
  imports: [EthModule, DBModule, ProofModule],
  services: [
    SyncService,
    GuardService,
    WalletService,
    JsonRpcService,
    OperatorService,
    ChainService,
  ],
})
export class AppModule {}
