import {
  getContractFactory,
  getContractInterface,
} from '@eth-optimism/contracts'
import { Watcher } from '@eth-optimism/core-utils'
import {
  Contract,
  Wallet,
  constants,
  providers,
  BigNumberish,
  BigNumber,
} from 'ethers'
import { Direction, waitForXDomainTransaction } from './watcher-utils'

export const GWEI = BigNumber.from(1e9)

// The hardhat instance
const l1HttpPort = 9545
export const l1Provider = new providers.JsonRpcProvider(
  `http://localhost:${l1HttpPort}`
)
l1Provider.pollingInterval = 10

const httpPort = 8545
export const l2Provider = new providers.JsonRpcProvider(
  `http://localhost:${httpPort}`
)
l2Provider.pollingInterval = 10

// The sequencer private key which is funded on L1
export const l1Wallet = new Wallet(
  '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
  l1Provider
)

// A random private key which should always be funded with deposits from L1 -> L2
// if it's using non-0 gas price
export const l2Wallet = l1Wallet.connect(l2Provider)

// Predeploys
export const PROXY_SEQUENCER_ENTRYPOINT_ADDRESS =
  '0x4200000000000000000000000000000000000004'
export const OVM_ETH_ADDRESS = '0x4200000000000000000000000000000000000006'

// The address manager is always at the same address in testnet deployments
export const addressManagerAddress =
  '0x5FbDB2315678afecb367f032d93F642f64180aa3'

export const getAddressManager = (provider: any) => {
  return getContractFactory('Lib_AddressManager')
    .connect(provider)
    .attach(addressManagerAddress)
}

// Gets the gateway using the proxy if available
export const getGateway = async (wallet: Wallet, AddressManager: Contract) => {
  const l1GatewayInterface = getContractInterface('OVM_L1ETHGateway')
  const ProxyGatewayAddress = await AddressManager.getAddress(
    'Proxy__OVM_L1ETHGateway'
  )
  const addressToUse =
    ProxyGatewayAddress !== constants.AddressZero
      ? ProxyGatewayAddress
      : await AddressManager.getAddress('OVM_L1ETHGateway')

  const OVM_L1ETHGateway = new Contract(
    addressToUse,
    l1GatewayInterface,
    wallet
  )

  return OVM_L1ETHGateway
}

export const getOvmEth = (wallet: Wallet) => {
  const OVM_ETH = new Contract(
    OVM_ETH_ADDRESS,
    getContractInterface('OVM_ETH'),
    wallet
  )

  return OVM_ETH
}

export const fundUser = async (
  watcher: Watcher,
  gateway: Contract,
  amount: BigNumberish,
  recipient?: string
) => {
  const value = BigNumber.from(amount)
  const tx = recipient
    ? gateway.depositTo(recipient, { value })
    : gateway.deposit({ value })
  await waitForXDomainTransaction(watcher, tx, Direction.L1ToL2)
}

export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
