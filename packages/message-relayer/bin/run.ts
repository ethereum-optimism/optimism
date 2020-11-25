import { Wallet } from 'ethers'
import { JsonRpcProvider } from '@ethersproject/providers'
import { MessageRelayerService } from '../src/service/message-relayer.service'

const env = process.env
const L2_NODE_WEB3_URL = env.L2_NODE_WEB3_URL
const L1_NODE_WEB3_URL = env.L1_NODE_WEB3_URL
const STATE_COMMITMENT_CHAIN_ADDRESS = env.STATE_COMMITMENT_CHAIN_ADDRESS
const L1_CROSS_DOMAIN_MESSENGER_ADDRESS = env.L1_CROSS_DOMAIN_MESSENGER_ADDRESS
const L2_CROSS_DOMAIN_MESSENGER_ADDRESS = env.L2_CROSS_DOMAIN_MESSENGER_ADDRESS
const L2_TO_L1_MESSAGE_PASSER_ADDRESS =
  env.L2_TO_L1_MESSAGE_PASSER_ADDRESS || '0x4200000000000000000000000000000000000000'
const POLLING_INTERVAL = env.POLLING_INTERVAL || '5000'
const RELAY_SIGNER = env.RELAY_SIGNER
const BLOCK_OFFSET = env.BLOCK_OFFSET || '1'
const L2_CHAIN_START_HEIGHT = env.L2_CHAIN_START_HEIGHT || '0'

const main = async () => {
  if (!STATE_COMMITMENT_CHAIN_ADDRESS) {
    throw new Error('Must pass STATE_COMMITMENT_CHAIN_ADDRESS')
  }
  if (!L1_CROSS_DOMAIN_MESSENGER_ADDRESS) {
    throw new Error('Must pass L1_CROSS_DOMAIN_MESSENGER_ADDRESS')
  }
  if (!L2_CROSS_DOMAIN_MESSENGER_ADDRESS) {
    throw new Error('Must pass L2_CROSS_DOMAIN_MESSENGER_ADDRESS')
  }

  const l2Provider = new JsonRpcProvider(L2_NODE_WEB3_URL)
  const l1Provider = new JsonRpcProvider(L1_NODE_WEB3_URL)

  const wallet = new Wallet(RELAY_SIGNER, l1Provider)

  const service = new MessageRelayerService({
    l1RpcProvider: l1Provider,
    l2RpcProvider: l2Provider,
    stateCommitmentChainAddress: STATE_COMMITMENT_CHAIN_ADDRESS,
    l1CrossDomainMessengerAddress: L1_CROSS_DOMAIN_MESSENGER_ADDRESS,
    l2CrossDomainMessengerAddress: L2_CROSS_DOMAIN_MESSENGER_ADDRESS,
    l2ToL1MessagePasserAddress: L2_TO_L1_MESSAGE_PASSER_ADDRESS,
    pollingInterval: parseInt(POLLING_INTERVAL, 10),
    relaySigner: wallet,
    l2ChainStartingHeight: L2_CHAIN_START_HEIGHT,
    blockOffset: parseInt(BLOCK_OFFSET, 10)
  })

  await service.start()
}

(async () => {
  await main()
})().catch(err => {
  console.log(err)
  process.exit(1)
})
