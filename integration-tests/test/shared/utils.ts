/* Imports: External */
import { Wallet, providers, BigNumber, utils } from 'ethers'
import { predeploys } from '@eth-optimism/contracts'
import { remove0x } from '@eth-optimism/core-utils'
import {
  CrossChainMessenger,
  NumberLike,
  asL2Provider,
} from '@eth-optimism/sdk'
import { cleanEnv, str, num, bool, makeValidator } from 'envalid'
import dotenv from 'dotenv'
dotenv.config()

/* Imports: Internal */
import { OptimismEnv } from './env'

export const isLiveNetwork = () => {
  return process.env.IS_LIVE_NETWORK === 'true'
}

export const HARDHAT_CHAIN_ID = 31337
export const DEFAULT_TEST_GAS_L1 = 330_000
export const DEFAULT_TEST_GAS_L2 = 1_300_000
export const ON_CHAIN_GAS_PRICE = 'onchain'

const gasPriceValidator = makeValidator((gasPrice) => {
  if (gasPrice === 'onchain') {
    return gasPrice
  }

  return num()._parse(gasPrice).toString()
})

const addressValidator = makeValidator((addr) => {
  if (!addr) {
    return ''
  } else if (utils.isAddress(addr)) {
    return addr
  } else {
    throw new Error('Expected an address')
  }
})

const procEnv = cleanEnv(process.env, {
  L1_GAS_PRICE: gasPriceValidator({
    default: '0',
  }),
  L1_URL: str({ default: 'http://localhost:9545' }),
  L1_POLLING_INTERVAL: num({ default: 10 }),

  L2_CHAINID: num({ default: 17 }),
  L2_GAS_PRICE: gasPriceValidator({
    default: 'onchain',
  }),
  L2_URL: str({ default: 'http://localhost:8545' }),
  L2_POLLING_INTERVAL: num({ default: 10 }),
  L2_WALLET_MIN_BALANCE_ETH: num({
    default: 2,
  }),
  L2_WALLET_TOP_UP_AMOUNT_ETH: num({
    default: 3,
  }),

  REPLICA_URL: str({ default: 'http://localhost:8549' }),
  REPLICA_POLLING_INTERVAL: num({ default: 10 }),

  VERIFIER_URL: str({ default: 'http://localhost:8547' }),

  HEALTHCHECK_URL: str({ default: 'http://localhost:7300/metrics' }),

  PRIVATE_KEY: str({
    default:
      '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
  }),
  GAS_PRICE_ORACLE_PRIVATE_KEY: str({
    default:
      '0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba',
  }),

  OVMCONTEXT_SPEC_NUM_TXS: num({
    default: 5,
  }),
  DTL_ENQUEUE_CONFIRMATIONS: num({
    default: 0,
  }),

  RUN_WITHDRAWAL_TESTS: bool({
    default: true,
  }),
  RUN_REPLICA_TESTS: bool({
    default: true,
  }),
  RUN_HEALTHCHECK_TESTS: bool({
    default: true,
  }),
  RUN_DEBUG_TRACE_TESTS: bool({
    default: true,
  }),
  RUN_STRESS_TESTS: bool({
    default: true,
  }),
  RUN_VERIFIER_TESTS: bool({
    default: true,
  }),
  RUN_SYSTEM_ADDRESS_TESTS: bool({
    default: false,
  }),

  MOCHA_TIMEOUT: num({
    default: 120_000,
  }),
  MOCHA_BAIL: bool({
    default: false,
  }),
  BATCH_SUBMITTER_SEQUENCER_BATCH_TYPE: str({
    default: 'zlib',
  }),

  ADDRESS_MANAGER: addressValidator({
    default: '',
  }),
  L1_CROSS_DOMAIN_MESSENGER: addressValidator({
    default: '',
  }),
  L1_STANDARD_BRIDGE: addressValidator({
    default: '',
  }),
  STATE_COMMITMENT_CHAIN: addressValidator({
    default: '',
  }),
  CANONICAL_TRANSACTION_CHAIN: addressValidator({
    default: '',
  }),
  BOND_MANAGER: addressValidator({
    default: '',
  }),
})

export const envConfig = procEnv

// The hardhat instance
export const l1Provider = new providers.JsonRpcProvider(procEnv.L1_URL)
l1Provider.pollingInterval = procEnv.L1_POLLING_INTERVAL

export const l2Provider = asL2Provider(
  new providers.JsonRpcProvider(procEnv.L2_URL)
)
l2Provider.pollingInterval = procEnv.L2_POLLING_INTERVAL

export const replicaProvider = asL2Provider(
  new providers.JsonRpcProvider(procEnv.REPLICA_URL)
)
replicaProvider.pollingInterval = procEnv.REPLICA_POLLING_INTERVAL

export const verifierProvider = asL2Provider(
  new providers.JsonRpcProvider(procEnv.VERIFIER_URL)
)
verifierProvider.pollingInterval = procEnv.L2_POLLING_INTERVAL

// The sequencer private key which is funded on L1
export const l1Wallet = new Wallet(procEnv.PRIVATE_KEY, l1Provider)

// A random private key which should always be funded with deposits from L1 -> L2
// if it's using non-0 gas price
export const l2Wallet = l1Wallet.connect(l2Provider)

// The owner of the GasPriceOracle on L2
export const gasPriceOracleWallet = new Wallet(
  procEnv.GAS_PRICE_ORACLE_PRIVATE_KEY,
  l2Provider
)

// Predeploys
export const OVM_ETH_ADDRESS = predeploys.OVM_ETH

export const L2_CHAINID = procEnv.L2_CHAINID

export const fundUser = async (
  messenger: CrossChainMessenger,
  amount: NumberLike,
  recipient?: string
) => {
  await messenger.waitForMessageReceipt(
    await messenger.depositETH(amount, {
      l2GasLimit: DEFAULT_TEST_GAS_L2,
      overrides: {
        gasPrice: DEFAULT_TEST_GAS_L1,
      },
    })
  )

  if (recipient !== undefined) {
    const tx = await messenger.l2Signer.sendTransaction({
      to: recipient,
      value: amount,
    })
    await tx.wait()
  }
}

export const conditionalTest = (
  condition: (env?: OptimismEnv) => Promise<boolean>,
  name,
  fn,
  message?: string,
  timeout?: number
) => {
  it(name, async function () {
    const shouldRun = await condition()
    if (!shouldRun) {
      console.log(message)
      this.skip()
      return
    }

    await fn()
  }).timeout(timeout || envConfig.MOCHA_TIMEOUT * 2)
}

export const withdrawalTest = (name, fn, timeout?: number) =>
  conditionalTest(
    () => Promise.resolve(procEnv.RUN_WITHDRAWAL_TESTS),
    name,
    fn,
    `Skipping withdrawal test.`,
    timeout
  )

export const hardhatTest = (name, fn) =>
  conditionalTest(
    isHardhat,
    name,
    fn,
    'Skipping test on non-Hardhat environment.'
  )

const abiCoder = new utils.AbiCoder()
export const encodeSolidityRevertMessage = (_reason: string): string => {
  return '0x08c379a0' + remove0x(abiCoder.encode(['string'], [_reason]))
}

export const defaultTransactionFactory = () => {
  return {
    to: '0x' + '1234'.repeat(10),
    gasLimit: 8_000_000,
    gasPrice: BigNumber.from(0),
    data: '0x',
    value: 0,
  }
}

export const gasPriceForL2 = async () => {
  if (procEnv.L2_GAS_PRICE === ON_CHAIN_GAS_PRICE) {
    return l2Wallet.getGasPrice()
  }

  return utils.parseUnits(procEnv.L2_GAS_PRICE, 'wei')
}

export const gasPriceForL1 = async () => {
  if (procEnv.L1_GAS_PRICE === ON_CHAIN_GAS_PRICE) {
    return l1Wallet.getGasPrice()
  }

  return utils.parseUnits(procEnv.L1_GAS_PRICE, 'wei')
}

export const isHardhat = async () => {
  const chainId = await l1Wallet.getChainId()
  return chainId === HARDHAT_CHAIN_ID
}

export const die = (...args) => {
  console.log(...args)
  process.exit(1)
}

export const logStderr = (msg: string) => {
  process.stderr.write(`${msg}\n`)
}
