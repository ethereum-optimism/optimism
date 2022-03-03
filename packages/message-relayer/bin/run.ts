import { Wallet, providers } from 'ethers'
import { Bcfg } from '@eth-optimism/core-utils'
import { Logger, LoggerOptions } from '@eth-optimism/common-ts'
import * as Sentry from '@sentry/node'
import * as dotenv from 'dotenv'
import Config from 'bcfg'

import { MessageRelayerService } from '../src'

dotenv.config()

const main = async () => {
  const config: Bcfg = new Config('message-relayer')
  config.load({
    env: true,
    argv: true,
  })

  const env = process.env

  const SENTRY_DSN = config.str('sentry-dsn', env.SENTRY_DSN)
  const USE_SENTRY = config.bool('use-sentry', env.USE_SENTRY === 'true')
  const ETH_NETWORK_NAME = config.str('eth-network-name', env.ETH_NETWORK_NAME)

  const loggerOptions: LoggerOptions = {
    name: 'Message_Relayer',
  }

  if (USE_SENTRY) {
    const sentryOptions = {
      release: `message-relayer@${process.env.npm_package_version}`,
      dsn: SENTRY_DSN,
      environment: ETH_NETWORK_NAME,
    }
    loggerOptions.sentryOptions = sentryOptions
    Sentry.init(sentryOptions)
  }

  const logger = new Logger(loggerOptions)

  const L2_NODE_WEB3_URL = config.str('l2-node-web3-url', env.L2_NODE_WEB3_URL)
  const L1_NODE_WEB3_URL = config.str('l1-node-web3-url', env.L1_NODE_WEB3_URL)
  const ADDRESS_MANAGER_ADDRESS = config.str(
    'address-manager-address',
    env.ADDRESS_MANAGER_ADDRESS
  )
  const L1_WALLET_KEY = config.str('l1-wallet-key', env.L1_WALLET_KEY)
  const MNEMONIC = config.str('mnemonic', env.MNEMONIC)
  const HD_PATH = config.str('hd-path', env.HD_PATH)
  const RELAY_GAS_LIMIT = config.uint(
    'relay-gas-limit',
    parseInt(env.RELAY_GAS_LIMIT, 10) || 4000000
  )
  const POLLING_INTERVAL = config.uint(
    'polling-interval',
    parseInt(env.POLLING_INTERVAL, 10) || 5000
  )
  const GET_LOGS_INTERVAL = config.uint(
    'get-logs-interval',
    parseInt(env.GET_LOGS_INTERVAL, 10) || 2000
  )
  const FROM_L2_TRANSACTION_INDEX = config.uint(
    'from-l2-transaction-index',
    parseInt(env.FROM_L2_TRANSACTION_INDEX, 10) || 0
  )

  if (!ADDRESS_MANAGER_ADDRESS) {
    throw new Error('Must pass ADDRESS_MANAGER_ADDRESS')
  }
  if (!L1_NODE_WEB3_URL) {
    throw new Error('Must pass L1_NODE_WEB3_URL')
  }
  if (!L2_NODE_WEB3_URL) {
    throw new Error('Must pass L2_NODE_WEB3_URL')
  }

  const l2Provider = new providers.StaticJsonRpcProvider({
    url: L2_NODE_WEB3_URL,
    headers: { 'User-Agent': 'message-relayer' },
  })
  const l1Provider = new providers.StaticJsonRpcProvider({
    url: L1_NODE_WEB3_URL,
    headers: { 'User-Agent': 'message-relayer' },
  })

  let wallet: Wallet
  if (L1_WALLET_KEY) {
    wallet = new Wallet(L1_WALLET_KEY, l1Provider)
  } else if (MNEMONIC) {
    wallet = Wallet.fromMnemonic(MNEMONIC, HD_PATH)
    wallet = wallet.connect(l1Provider)
  } else {
    throw new Error('Must pass one of L1_WALLET_KEY or MNEMONIC')
  }

  const service = new MessageRelayerService({
    l2RpcProvider: l2Provider,
    l1Wallet: wallet,
    relayGasLimit: RELAY_GAS_LIMIT,
    fromL2TransactionIndex: FROM_L2_TRANSACTION_INDEX,
    pollingInterval: POLLING_INTERVAL,
    getLogsInterval: GET_LOGS_INTERVAL,
    logger,
  })

  await service.start()
}

main()
