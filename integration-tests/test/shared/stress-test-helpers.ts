/* Imports: External */
import { ethers } from 'ethers'
import { sleep } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { OptimismEnv } from './env'
import { gasPriceForL1, gasPriceForL2 } from './utils'

interface TransactionParams {
  contract: ethers.Contract
  functionName: string
  functionParams: any[]
}

// Arbitrary big amount of gas for the L1<>L2 messages.
const MESSAGE_GAS = 8_000_000

export const fundRandomWallet = async (
  env: OptimismEnv,
  wallet: ethers.Wallet,
  value: ethers.BigNumber
): Promise<ethers.Wallet> => {
  const fundTx = await env.l1Wallet.sendTransaction({
    gasLimit: 25_000,
    to: wallet.address,
    gasPrice: await gasPriceForL1(),
    value,
  })
  await fundTx.wait()
  return wallet
}

export const executeL1ToL2Transaction = async (
  env: OptimismEnv,
  wallet: ethers.Wallet,
  tx: TransactionParams
) => {
  const signer = wallet.connect(env.l1Wallet.provider)
  const receipt = await retryOnNonceError(async () =>
    env.messenger.contracts.l1.L1CrossDomainMessenger.connect(
      signer
    ).sendMessage(
      tx.contract.address,
      tx.contract.interface.encodeFunctionData(
        tx.functionName,
        tx.functionParams
      ),
      MESSAGE_GAS,
      {
        gasPrice: await gasPriceForL1(),
      }
    )
  )
  await env.waitForXDomainTransaction(receipt)
}

export const executeL2ToL1Transaction = async (
  env: OptimismEnv,
  wallet: ethers.Wallet,
  tx: TransactionParams
) => {
  const signer = wallet.connect(env.l2Wallet.provider)
  const receipt = await retryOnNonceError(() =>
    env.messenger.contracts.l2.L2CrossDomainMessenger.connect(
      signer
    ).sendMessage(
      tx.contract.address,
      tx.contract.interface.encodeFunctionData(
        tx.functionName,
        tx.functionParams
      ),
      MESSAGE_GAS,
      {
        gasPrice: gasPriceForL2(),
      }
    )
  )

  await env.relayXDomainMessages(receipt)
  await env.waitForXDomainTransaction(receipt)
}

export const executeL2Transaction = async (
  env: OptimismEnv,
  wallet: ethers.Wallet,
  tx: TransactionParams
) => {
  const signer = wallet.connect(env.l2Wallet.provider)
  const result = await retryOnNonceError(() =>
    tx.contract
      .connect(signer)
      .functions[tx.functionName](...tx.functionParams, {
        gasPrice: gasPriceForL2(),
      })
  )
  await result.wait()
}

export const executeRepeatedL1ToL2Transactions = async (
  env: OptimismEnv,
  wallets: ethers.Wallet[],
  tx: TransactionParams
) => {
  for (const wallet of wallets) {
    await executeL1ToL2Transaction(env, wallet, tx)
  }
}

export const executeRepeatedL2ToL1Transactions = async (
  env: OptimismEnv,
  wallets: ethers.Wallet[],
  tx: TransactionParams
) => {
  for (const wallet of wallets) {
    await executeL2ToL1Transaction(env, wallet, tx)
  }
}

export const executeRepeatedL2Transactions = async (
  env: OptimismEnv,
  wallets: ethers.Wallet[],
  tx: TransactionParams
) => {
  for (const wallet of wallets) {
    await executeL2Transaction(env, wallet, tx)
  }
}

export const executeL1ToL2TransactionsParallel = async (
  env: OptimismEnv,
  wallets: ethers.Wallet[],
  tx: TransactionParams
) => {
  await Promise.all(wallets.map((w) => executeL1ToL2Transaction(env, w, tx)))
}

export const executeL2ToL1TransactionsParallel = async (
  env: OptimismEnv,
  wallets: ethers.Wallet[],
  tx: TransactionParams
) => {
  await Promise.all(wallets.map((w) => executeL2ToL1Transaction(env, w, tx)))
}

export const executeL2TransactionsParallel = async (
  env: OptimismEnv,
  wallets: ethers.Wallet[],
  tx: TransactionParams
) => {
  await Promise.all(wallets.map((w) => executeL2Transaction(env, w, tx)))
}

const retryOnNonceError = async (cb: () => Promise<any>): Promise<any> => {
  while (true) {
    try {
      return await cb()
    } catch (err) {
      const msg = err.message.toLowerCase()

      if (
        msg.includes('nonce too low') ||
        msg.includes('nonce has already been used') ||
        msg.includes('transaction was replaced') ||
        msg.includes('another transaction with same nonce in the queue') ||
        msg.includes('reverted without a reason')
      ) {
        console.warn('Retrying transaction after nonce error.')
        await sleep(5000)
        continue
      }

      throw err
    }
  }
}
