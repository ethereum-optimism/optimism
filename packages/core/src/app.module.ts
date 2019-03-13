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
  EventModule,
  DBModule,
  ProofModule,
} from './services'

@Module({
  imports: [EthModule, EventModule, DBModule, ProofModule],
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
