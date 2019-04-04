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
import { DefaultMessageBus } from '../common/app/message-bus';
import { DefaultConfigManager } from '../common/app/config-manager';
import { DefaultLogCollector } from '../common/app/log-collector';
import { DefaultDBManager } from '../common/db/db-manager';
import { DefaultEthClient } from '../common';
import { DummyContractDetector } from './eth/dummy-contract-detector';

@Module({
  services: [
    DefaultMessageBus,
    DefaultConfigManager,
    DefaultLogCollector,
    DefaultDBManager,
    DefaultEthClient,
    ChainDbHost,
    {
      provide: PlasmaContractDetector,
      useClass: DummyContractDetector,
    },
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
