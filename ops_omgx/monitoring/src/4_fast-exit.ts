import { Contract, providers, utils, Wallet } from 'ethers'
import { getContractFactory, getContractInterface } from '@eth-optimism/contracts'
import {
  initFastWatcher,
  Direction,
  waitForXDomainTransaction,
} from './libs/watcher-utils'

import L2DepositedERC20Json from './artifacts-ovm/contracts/L2DepositedERC20.sol/L2DepositedERC20.json'
import L2LiquidityPoolJson from './artifacts-ovm/contracts/LP/L2LiquidityPool.sol/L2LiquidityPool.json'

import logger from './logger'

export const PROXY_SEQUENCER_ENTRYPOINT_ADDRESS = '0x4200000000000000000000000000000000000004'
export const OVM_ETH_ADDRESS = '0x4200000000000000000000000000000000000006'
// tslint:disable-next-line: variable-name
export const Proxy__OVM_L2CrossDomainMessenger = '0x4200000000000000000000000000000000000007'
export const addressManagerAddress = process.env.L1_ADDRESS_MANAGER

const walletPKey = process.env.WALLET_PRIVATE_KEY
const l2PoolAddress = process.env.L2_LIQUIDITY_POOL_ADDRESS
const l1Web3Url = process.env.L1_NODE_WEB3_URL
const l2Web3Url = process.env.L2_NODE_WEB3_URL
const l2DepositedERC20 = process.env.L2_DEPOSITED_ERC20
const dummyEthAmount = process.env.DUMMY_ETH_AMOUNT
const l1Provider = new providers.JsonRpcProvider(l1Web3Url)
const l2Provider = new providers.JsonRpcProvider(l2Web3Url)
const l1Wallet = new Wallet(walletPKey, l1Provider)
const l2Wallet = l1Wallet.connect(l2Provider)

const getAddressManager = (provider: any) => {
  return getContractFactory('Lib_AddressManager')
    .connect(provider)
    .attach(addressManagerAddress) as any
}

const getL2ETHGateway = (wallet: Wallet) => {
  const OVM_ETH = new Contract(
    OVM_ETH_ADDRESS,
    getContractInterface('OVM_ETH') as any,
    wallet
  )
  return OVM_ETH
}

export const fastExit = async () => {
  const l1Address = await l1Wallet.getAddress()
  const l2Address = await l2Wallet.getAddress()
  const addressManager = getAddressManager(l1Wallet)
  const watcher = await initFastWatcher(l1Provider, l2Provider, addressManager)

  const L2LiquidityPool = new Contract(
    l2PoolAddress,
    L2LiquidityPoolJson.abi,
    l2Wallet
  )

  const L2DepositedERC20 = new Contract(
    l2DepositedERC20,
    L2DepositedERC20Json.abi,
    l2Wallet
  )

  const L2ETHGateway = getL2ETHGateway(l2Wallet)
  const fastExitAmount = utils.parseEther(dummyEthAmount)

  const l1Balance = await l1Provider.getBalance(l1Address)
  const l2Balance = await l2Provider.getBalance(l2Address)
  const l2ERCBalance = await L2DepositedERC20.balanceOf(l2Wallet.address)
  logger.info('Start dummy transfer from L2->L1', {
    l1Address,
    l2Address,
    l1Balance: utils.formatEther(l1Balance),
    l2Balance: utils.formatEther(l2Balance),
    L2ERCBalance: utils.formatEther(l2ERCBalance)
  })

  logger.info('Approve TX')
  const approveL2TX = await L2ETHGateway.connect(l2Wallet).approve(
    L2LiquidityPool.address,
    fastExitAmount,
    { gasLimit: 800000, gasPrice: 0 }
  )
  await approveL2TX.wait()
  logger.info('Approve TX... Done')

  logger.info('Cross Domain Fast Exit')
  await waitForXDomainTransaction(
    watcher,
    L2LiquidityPool.connect(l2Wallet).clientDepositL2(
      fastExitAmount,
      L2ETHGateway.address,
      { gasLimit: 3000000, gasPrice: 0 }
    ),
    Direction.L2ToL1
  )
  logger.info('Cross Domain Fast Exit...Done')

  const l1BalanceAfter = await l1Provider.getBalance(l1Address)
  const l2BalanceAfter = await l2Provider.getBalance(l2Address)
  const l2ERCBalanceAfter = await L2DepositedERC20.balanceOf(l2Wallet.address)
  logger.info('Done dummy transfer from L2->L1', {
    l1Address,
    l2Address,
    l1Balance: utils.formatEther(l1BalanceAfter),
    l2Balance: utils.formatEther(l2BalanceAfter),
    l2ERCBalance: utils.formatEther(l2ERCBalanceAfter),
  })
}
