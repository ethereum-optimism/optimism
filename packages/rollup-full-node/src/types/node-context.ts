import { JsonRpcProvider, Provider } from 'ethers/providers'
import { Contract, Wallet } from 'ethers'

export interface L1NodeContext {
  provider: Provider
  sequencerWallet: Wallet
  l2ToL1MessageReceiver: Contract
  l1ToL2TransactionPasser: Contract
}

export interface L2NodeContext {
  provider: JsonRpcProvider
  wallet: Wallet
  executionManager: Contract
  l2ToL1MessagePasser: Contract
}
