/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import {
  SyncService,
  GuardService,
  WalletService,
  JSONRPCService,
  EventWatcherService,
  EventHandlerService,
  EthService,
  OperatorService,
  DBService,
  ChainService,
  ProofService,
} from './services'

@Module({
  services: [
    SyncService,
    GuardService,
    WalletService,
    JSONRPCService,
    EventWatcherService,
    EventHandlerService,
    EthService,
    OperatorService,
    DBService,
    ChainService,
    ProofService,
  ],
})
export class AppModule {}
