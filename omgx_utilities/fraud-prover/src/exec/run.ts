import { Wallet, providers } from 'ethers'
import { FraudProverService } from '../service'
import { Bcfg } from '@eth-optimism/core-utils'
import * as dotenv from 'dotenv'
import Config from 'bcfg'

dotenv.config()

const main = async () => {
  const config: Bcfg = new Config('fraud-prover')
  config.load({
    env: true,
    argv: true,
  })

  const env = process.env
  const L1_NODE_WEB3_URL = config.str('l1-node-web3-url', env.L1_NODE_WEB3_URL)
  const L2_NODE_WEB3_URL = config.str(
    'verifier-web3-url',
    env.VERIFIER_WEB3_URL
  )

  const ADDRESS_MANAGER_ADDRESS = config.str(
    'address-manager-address',
    env.ADDRESS_MANAGER_ADDRESS
  )
  const FP_WALLET_KEY = config.str('fp-wallet-key', env.FP_WALLET_KEY)
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
  const L2_BLOCK_OFFSET = config.uint(
    'l2-start-offset',
    parseInt(env.L2_BLOCK_OFFSET, 10) || 1
  )
  const L1_START_OFFSET = config.uint(
    'l1-start-offset',
    parseInt(env.L1_BLOCK_OFFSET, 10) || 1
  )
  const FROM_L2_TRANSACTION_INDEX = config.uint(
    'from-l2-transaction-index',
    parseInt(env.FROM_L2_TRANSACTION_INDEX, 10) || 0
  )
  const RUN_GAS_LIMIT = config.uint(
    'run-gas-limit',
    parseInt(env.RUN_GAS_LIMIT, 10) || 95000000
  )
  const L1_BLOCK_FINALITY = config.uint(
    'l1-block-finality',
    parseInt(env.L1_BLOCK_FINALITY, 10) || 0
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

  console.log('The L2 block offset is:', L2_BLOCK_OFFSET)

  const l2Provider = new providers.JsonRpcProvider(L2_NODE_WEB3_URL)
  const l1Provider = new providers.JsonRpcProvider(L1_NODE_WEB3_URL)

  let wallet: Wallet
  if (FP_WALLET_KEY) {
    wallet = new Wallet(FP_WALLET_KEY, l1Provider)
  } else if (MNEMONIC) {
    wallet = Wallet.fromMnemonic(MNEMONIC, HD_PATH)
    wallet = wallet.connect(l1Provider)
  } else {
    throw new Error('Must pass one of FP_WALLET_KEY or MNEMONIC')
  }

  const service = new FraudProverService({
    l1RpcProvider: l1Provider,
    l2RpcProvider: l2Provider,
    addressManagerAddress: ADDRESS_MANAGER_ADDRESS,
    l1Wallet: wallet,
    deployGasLimit: RELAY_GAS_LIMIT, //should reconcile naming
    runGasLimit: RUN_GAS_LIMIT, //should reconcile naming
    fromL2TransactionIndex: FROM_L2_TRANSACTION_INDEX,
    pollingInterval: POLLING_INTERVAL,
    l2BlockOffset: L2_BLOCK_OFFSET,
    l1StartOffset: L1_START_OFFSET,
    l1BlockFinality: L1_BLOCK_FINALITY,
    //getLogsInterval: GET_LOGS_INTERVAL,
  })

  await service.start()
}
export default main
