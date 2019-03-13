/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import { DBService } from './db.service'
import { EphemDBProvider } from './backends/ephem-provider'
import { ChainDB } from './interfaces/chain-db'
import { SyncDB } from './interfaces/sync-db'
import { WalletDB } from './interfaces/wallet-db'

@Module({
  services: [DBService, EphemDBProvider, ChainDB, SyncDB, WalletDB],
})
export class DBModule {}
