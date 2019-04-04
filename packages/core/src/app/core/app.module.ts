import { Module } from '@nestd/core'
import { ChainDbHost } from './db'
import {
  PlasmaContractDetector,
  KeyManagerHost,
  PlasmaContractHost,
} from './eth'
import { DefaultRpcClient, DefaultRpcServer } from './networking'
import {
  PGHistoryManagerHost,
  PGStateManagerHost,
  DefaultTransactionReceiver,
  DefaultTransactionWatcher,
} from './state'

@Module({
  services: [
    ChainDbHost,
    PlasmaContractDetector,
    KeyManagerHost,
    PlasmaContractHost,
    DefaultRpcClient,
    DefaultRpcServer,
    PGHistoryManagerHost,
    PGStateManagerHost,
    DefaultTransactionReceiver,
    DefaultTransactionWatcher,
  ],
})
export class CoreAppModule {}
