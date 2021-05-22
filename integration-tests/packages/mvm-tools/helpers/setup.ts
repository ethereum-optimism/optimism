import { JsonRpcProvider, TransactionReceipt, TransactionResponse } from '@ethersproject/providers'
import { Contract, Wallet } from 'ethers'
import { Config } from '../../../common'

import { getContractInterface, getContractFactory } from '@eth-optimism/contracts'
import { Watcher } from '@eth-optimism/watcher'
import { Transaction } from '@ethersproject/transactions'

export const getEnvironment = async (): Promise<{
    l1Provider: JsonRpcProvider,
    l2Provider: JsonRpcProvider,
    l1Wallet: Wallet,
    l2Wallet: Wallet,
    AddressManager: Contract,
    watcher: Watcher
}> => {
    const l1Provider = new JsonRpcProvider(Config.L1NodeUrlWithPort())
    const l2Provider = new JsonRpcProvider(Config.L2NodeUrlWithPort())
    const l1Wallet = new Wallet(Config.DeployerPrivateKey(), l1Provider)
    const l2Wallet = new Wallet(Config.DeployerPrivateKey(), l2Provider)

    const addressManagerAddress = Config.AddressResolverAddress()
    const addressManagerInterface = getContractInterface('Lib_AddressManager')
    const AddressManager = new Contract(addressManagerAddress, addressManagerInterface, l1Provider)

    const watcher = await initWatcher(
        l1Provider,
        l2Provider,
        AddressManager
    )

    return {
        l1Provider,
        l2Provider,
        l1Wallet,
        l2Wallet,
        AddressManager,
        watcher
    }
}

export const initWatcher = async (
    l1Provider: JsonRpcProvider,
    l2Provider: JsonRpcProvider,
    AddressManager: Contract
) => {
    const l1MessengerAddress = await AddressManager.getAddress('Proxy__OVM_L1CrossDomainMessenger')
    const l2MessengerAddress = await AddressManager.getAddress('OVM_L2CrossDomainMessenger')
    return new Watcher({
      l1: {
        provider: l1Provider,
        messengerAddress: l1MessengerAddress
      },
      l2: {
        provider: l2Provider,
        messengerAddress: l2MessengerAddress
      }
    })
  }

interface CrossDomainMessagePair {
    l1tx: Transaction,
    l1receipt: TransactionReceipt,
    l2tx: Transaction,
    l2receipt: TransactionReceipt
  }

export const waitForDepositTypeTransaction = async (
    l1OriginatingTx: Promise<TransactionResponse>,
    watcher: Watcher,
    l1Provider: JsonRpcProvider,
    l2Provider: JsonRpcProvider
): Promise<CrossDomainMessagePair> => {
    const res = await l1OriginatingTx
    await res.wait()

    const l1tx = await l1Provider.getTransaction(res.hash)
    const l1receipt = await l1Provider.getTransactionReceipt(res.hash)
    const [l1ToL2XDomainMsgHash] = await watcher.getMessageHashesFromL1Tx(res.hash)
    const l2receipt = await watcher.getL2TransactionReceipt(l1ToL2XDomainMsgHash) as TransactionReceipt
    const l2tx = await l2Provider.getTransaction(l2receipt.transactionHash)

    return {
        l1tx,
        l1receipt,
        l2tx,
        l2receipt
    }
}

// TODO: combine these elegantly? v^v^v
export const waitForWithdrawalTypeTransaction = async (
    l2OriginatingTx: Promise<TransactionResponse>,
    watcher: Watcher,
    l1Provider: JsonRpcProvider,
    l2Provider: JsonRpcProvider
  ): Promise<CrossDomainMessagePair> => {
    const res = await l2OriginatingTx
    await res.wait()

    const l2tx = await l2Provider.getTransaction(res.hash)
    const l2receipt = await l2Provider.getTransactionReceipt(res.hash)
    const [l2ToL1XDomainMsgHash] = await watcher.getMessageHashesFromL2Tx(res.hash)
    const l1receipt = await watcher.getL1TransactionReceipt(l2ToL1XDomainMsgHash) as TransactionReceipt
    const l1tx = await l1Provider.getTransaction(l1receipt.transactionHash)

    return {
      l2tx,
      l2receipt,
      l1tx,
      l1receipt
    }
  }
