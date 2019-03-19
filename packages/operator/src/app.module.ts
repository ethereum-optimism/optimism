/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import { RpcServerService } from './services/rpc-server.service'

@Module({
  services: [RpcServerService],
})
export class OperatorAppModule {}
