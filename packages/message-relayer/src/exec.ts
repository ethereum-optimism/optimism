/* Imports: External */
import { Bcfg } from '@eth-optimism/core-utils'
import { predeploys } from '@eth-optimism/contracts'
import * as dotenv from 'dotenv'
import Config from 'bcfg'

/* Imports: Internal */
import { MessageRelayerService } from './service'

// Load environment variables from .env
dotenv.config()

const main = async () => {
  const config: Bcfg = new Config('message-relayer')
  config.load({
    env: true,
    argv: true,
  })

  const l1RpcProviderUrl = config.str('l1-rpc-provider-url')
  const l2RpcProviderUrl = config.str('l2-rpc-provider-url')
  const stateCommitmentChainAddress = config.str(
    'state-commitment-chain-address'
  )
  const l1CrossDomainMessengerAddress = config.str(
    'l1-cross-domain-messenger-address'
  )
  const l2CrossDomainMessengerAddress = predeploys.OVM_L2CrossDomainMessenger
  const relayerPrivateKey = config.str('relayer-private-key')
  const pollingIntervalMs = config.uint('polling-interval-ms')

  const service = new MessageRelayerService({
    l1RpcProvider: l1RpcProviderUrl,
    l2RpcProvider: l2RpcProviderUrl,
    stateCommitmentChainAddress,
    l1CrossDomainMessengerAddress,
    l2CrossDomainMessengerAddress,
    relayerPrivateKey,
    pollingIntervalMs,
  })

  await service.start()
}

main()
