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

  const GAS_PRICE_ORACLE_OWNER_PRIVATE_KEY = config.str(
    'gas-price-oracle-owner-key',
    env.GAS_PRICE_ORACLE_OWNER_PRIVATE_KEY
  )
  const SEQUENCER_PRIVATE_KEY = config.str(
    'sequencer-private-key',
    env.SEQUENCER_PRIVATE_KEY
  )
  const SEQUENCER_ADDRESS = config.str(
    'sequencer-address',
    env.SEQUENCER_ADDRESS
  )
  const PROPOSER_PRIVATE_KEY = config.str(
    'proposer-private-key',
    env.PROPOSER_PRIVATE_KEY
  )
  const PROPOSER_ADDRESS = config.str(
    'proposer-address',
    env.PROPOSER_ADDRESS
  )
  const RELAYER_PRIVATE_KEY = config.str(
    'relayer-private-key',
    env.RELAYER_PRIVATE_KEY
  )
  const RELAYER_ADDRESS = config.str(
    'relayer-address',
    env.RELAYER_ADDRESS
  )
  const FAST_RELAYER_PRIVATE_KEY = config.str(
    'fast-relayer-private-key',
    env.FAST_RELAYER_PRIVATE_KEY
  )
  const FAST_RELAYER_ADDRESS = config.str(
    'fast-relayer-address',
    env.FAST_RELAYER_ADDRESS
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

  if (!GAS_PRICE_ORACLE_ADDRESS) {
    throw new Error('Must pass GAS_PRICE_ORACLE_ADDRESS')
  }
  if (!L1_NODE_WEB3_URL) {
    throw new Error('Must pass L1_NODE_WEB3_URL')
  }
  if (!L2_NODE_WEB3_URL) {
    throw new Error('Must pass L2_NODE_WEB3_URL')
  }
  if (!GAS_PRICE_ORACLE_OWNER_PRIVATE_KEY) {
    throw new Error('Must pass GAS_PRICE_ORACLE_OWNER_PRIVATE_KEY')
  }
  if (!SEQUENCER_ADDRESS && !SEQUENCER_PRIVATE_KEY) {
    throw new Error('Must pass SEQUENCER_ADDRESS or SEQUENCER_PRIVATE_KEY')
  }
  if (!PROPOSER_ADDRESS && !PROPOSER_PRIVATE_KEY) {
    throw new Error('Must pass PROPOSER_ADDRESS or PROPOSER_PRIVATE_KEY')
  }
  if (!RELAYER_ADDRESS && !RELAYER_PRIVATE_KEY) {
    throw new Error('Must pass RELAYER_ADDRESS or RELAYER_PRIVATE_KEY')
  }
  if (!FAST_RELAYER_ADDRESS && !FAST_RELAYER_PRIVATE_KEY) {
    throw new Error('Must pass FAST_RELAYER_ADDRESS or FAST_RELAYER_PRIVATE_KEY')
  }

  const l1Provider = new providers.JsonRpcProvider(L1_NODE_WEB3_URL)
  const l2Provider = new providers.JsonRpcProvider(L2_NODE_WEB3_URL)

  const gasPriceOracleOwnerWallet = new Wallet(GAS_PRICE_ORACLE_OWNER_PRIVATE_KEY, l2Provider)

  // Fixed address
  const OVM_oETHAddress = "0x4200000000000000000000000000000000000006"
  const OVM_SequencerFeeVault = "0x4200000000000000000000000000000000000011"

  // sequencer, proposer, relayer and fast relayer addresses
  const sequencerAddress = SEQUENCER_ADDRESS ? SEQUENCER_ADDRESS:
    (new Wallet(SEQUENCER_PRIVATE_KEY, l2Provider)).address;
  const proposerAddress = PROPOSER_ADDRESS ? PROPOSER_ADDRESS:
    (new Wallet(PROPOSER_PRIVATE_KEY, l2Provider)).address;
  const relayerAddress = RELAYER_ADDRESS ? RELAYER_ADDRESS:
    (new Wallet(RELAYER_PRIVATE_KEY, l2Provider)).address;
  const fastRelayerAddress = FAST_RELAYER_ADDRESS ? FAST_RELAYER_ADDRESS:
    (new Wallet(FAST_RELAYER_PRIVATE_KEY, l2Provider)).address;

  const service = new GasPriceOracleService({
    l1RpcProvider: l1Provider,
    l2RpcProvider: l2Provider,
    gasPriceOracleAddress: GAS_PRICE_ORACLE_ADDRESS,
    OVM_oETHAddress,
    OVM_SequencerFeeVault,
    gasPriceOracleOwnerWallet,
    sequencerAddress,
    proposerAddress,
    relayerAddress,
    fastRelayerAddress,
    gasFloorPrice: GAS_PRICE_ORACLE_FLOOR_PRICE,
    gasRoofPrice: GAS_PRICE_ORACLE_ROOF_PRICE,
    gasPriceMinPercentChange: GAS_PRICE_ORACLE_MIN_PERCENT_CHANGE,
    pollingInterval: POLLING_INTERVAL
  })

  await service.start()
}
export default main
