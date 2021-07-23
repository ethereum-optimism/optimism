import { Wallet, providers } from 'ethers'
import { Bcfg } from '@eth-optimism/core-utils'
import * as dotenv from 'dotenv'
import Config from 'bcfg'

import { GasPriceOracleService } from '../service';

dotenv.config()

const main = async () => {
  const config: Bcfg = new Config('gas-price-oracle')
  config.load({
    env: true,
    argv: true,
  })

  const env = process.env
  const L2_NODE_WEB3_URL = config.str('l2-node-web3-url', env.L2_NODE_WEB3_URL)
  const L1_NODE_WEB3_URL = config.str('l1-node-web3-url', env.L1_NODE_WEB3_URL)

  const DEPLOYER_PRIVATE_KEY = config.str(
    'deployer-private-key',
    env.DEPLOYER_PRIVATE_KEY
  )
  const SEQUENCER_PRIVATE_KEY = config.str(
    'sequencer-private-key',
    env.SEQUENCER_PRIVATE_KEY
  )
  const PROPOSER_PRIVATE_KEY = config.str(
    'proposer-private-key',
    env.PROPOSER_PRIVATE_KEY
  )
  const RELAYER_PRIVATE_KEY = config.str(
    'relayer-private-key',
    env.RELAYER_PRIVATE_KEY
  )
  const FAST_RELAYER_PRIVATE_KEY = config.str(
    'fast-relayer-private-key',
    env.FAST_RELAYER_PRIVATE_KEY
  )

  const GAS_PRICE_ORACLE_ADDRESS = config.str(
    'gas-price-oracle',
    env.GAS_PRICE_ORACLE_ADDRESS
  )
  // 0.015 GWEI
  const GAS_PRICE_ORACLE_FLOOR_PRICE = config.uint(
    'gas-price-oracle-floor-price',
    parseInt(env.GAS_PRICE_ORACLE_FLOOR_PRICE, 10) || 150000
  )
  // 2 GWEI
  const GAS_PRICE_ORACLE_ROOF_PRICE = config.uint(
    'gas-price-oracle-roof-price',
    parseInt(env.GAS_PRICE_ORACLE_ROOF_PRICE, 10) || 20000000
  )
  const GAS_PRICE_ORACLE_MIN_PERCENT_CHANGE = config.uint(
    'gas-price-oracle-min-percent-change',
    parseFloat(env.GAS_PRICE_ORACLE_MIN_PERCENT_CHANGE) || 0.1
  )
  const POLLING_INTERVAL = config.uint(
    'polling-interval',
    parseInt(env.POLLING_INTERVAL, 10) || 1000 * 60 * 10
  )

  const ETHERSCAN_API = config.str(
    'etherscan-api',
    env.ETHERSCAN_API,
  )

  if (!GAS_PRICE_ORACLE_ADDRESS) {
    throw new Error('Must pass GAS_PRICE_ORACLE_ADDRESS')
  }
  if (!L1_NODE_WEB3_URL) {
    throw new Error('Must pass L1_NODE_WEB3_URL')
  }
  if (!L2_NODE_WEB3_URL) {
    throw new Error('Must pass L2_NODE_WEB3_URL')
  }
  if (!DEPLOYER_PRIVATE_KEY) {
    throw new Error('Must pass DEPLOYER_PRIVATE_KEY')
  }
  if (!SEQUENCER_PRIVATE_KEY) {
    throw new Error('Must pass SEQUENCER_PRIVATE_KEY')
  }
  if (!PROPOSER_PRIVATE_KEY) {
    throw new Error('Must pass PROPOSER_PRIVATE_KEY')
  }
  if (!RELAYER_PRIVATE_KEY) {
    throw new Error('Must pass RELAYER_PRIVATE_KEY')
  }
  if (!FAST_RELAYER_PRIVATE_KEY) {
    throw new Error('Must pass FAST_RELAYER_PRIVATE_KEY')
  }

  const l1Provider = new providers.JsonRpcProvider(L1_NODE_WEB3_URL)
  const l2Provider = new providers.JsonRpcProvider(L2_NODE_WEB3_URL)

  const deployerWallet = new Wallet(DEPLOYER_PRIVATE_KEY, l2Provider)
  const sequencerWallet = new Wallet(SEQUENCER_PRIVATE_KEY, l1Provider)
  const proposerWallet = new Wallet(PROPOSER_PRIVATE_KEY, l1Provider)
  const relayerWallet = new Wallet(RELAYER_PRIVATE_KEY, l1Provider)
  const fastRelayerWallet = new Wallet(FAST_RELAYER_PRIVATE_KEY, l1Provider)

  const service = new GasPriceOracleService({
    l1RpcProvider: l1Provider,
    l2RpcProvider: l2Provider,
    gasPriceOracleAddress: GAS_PRICE_ORACLE_ADDRESS,
    deployerWallet,
    sequencerWallet,
    proposerWallet,
    relayerWallet,
    fastRelayerWallet,
    gasFloorPrice: GAS_PRICE_ORACLE_FLOOR_PRICE,
    gasRoofPrice: GAS_PRICE_ORACLE_ROOF_PRICE,
    gasPriceMinPercentChange: GAS_PRICE_ORACLE_MIN_PERCENT_CHANGE,
    pollingInterval: POLLING_INTERVAL,
    etherscanAPI: ETHERSCAN_API,
  })

  await service.start()
}
export default main
