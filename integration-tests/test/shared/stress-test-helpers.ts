/* Imports: External */
import { ethers } from 'ethers'

/* Imports: Internal */
import { OptimismEnv } from './env'
import { Direction } from './watcher-utils'

interface TransactionParams {
  contract: ethers.Contract
  functionName: string
  functionParams: any[]
}

// Arbitrary big amount of gas for the L1<>L2 messages.
const MESSAGE_GAS = 8_000_000

export const executeL1ToL2Transactions = async (
  env: OptimismEnv,
  txs: TransactionParams[]
) => {
  for (const tx of txs) {
    const signer = ethers.Wallet.createRandom().connect(env.l1Wallet.provider)
    const receipt = await env.l1Messenger
      .connect(signer)
      .sendMessage(
        tx.contract.address,
        tx.contract.interface.encodeFunctionData(
          tx.functionName,
          tx.functionParams
        ),
        MESSAGE_GAS,
        {
          gasPrice: 0,
        }
      )

    await env.waitForXDomainTransaction(receipt, Direction.L1ToL2)
  }
}

export const executeL2ToL1Transactions = async (
  env: OptimismEnv,
  txs: TransactionParams[]
) => {
  for (const tx of txs) {
    const signer = ethers.Wallet.createRandom().connect(env.l2Wallet.provider)
    const receipt = await env.l2Messenger
      .connect(signer)
      .sendMessage(
        tx.contract.address,
        tx.contract.interface.encodeFunctionData(
          tx.functionName,
          tx.functionParams
        ),
        MESSAGE_GAS,
        {
          gasPrice: 0,
        }
      )

    await env.relayXDomainMessages(receipt)
    await env.waitForXDomainTransaction(receipt, Direction.L2ToL1)
  }
}

export const executeL2Transactions = async (
  env: OptimismEnv,
  txs: TransactionParams[]
) => {
  for (const tx of txs) {
    const signer = ethers.Wallet.createRandom().connect(env.l2Wallet.provider)
    const result = await tx.contract
      .connect(signer)
      .functions[tx.functionName](...tx.functionParams, {
        gasPrice: 0,
      })
    await result.wait()
  }
}

export const executeRepeatedL1ToL2Transactions = async (
  env: OptimismEnv,
  tx: TransactionParams,
  count: number
) => {
  await executeL1ToL2Transactions(
    env,
    [...Array(count).keys()].map(() => tx)
  )
}

export const executeRepeatedL2ToL1Transactions = async (
  env: OptimismEnv,
  tx: TransactionParams,
  count: number
) => {
  await executeL2ToL1Transactions(
    env,
    [...Array(count).keys()].map(() => tx)
  )
}

export const executeRepeatedL2Transactions = async (
  env: OptimismEnv,
  tx: TransactionParams,
  count: number
) => {
  await executeL2Transactions(
    env,
    [...Array(count).keys()].map(() => tx)
  )
}

export const executeL1ToL2TransactionsParallel = async (
  env: OptimismEnv,
  txs: TransactionParams[]
) => {
  await Promise.all(
    txs.map(async (tx) => {
      const signer = ethers.Wallet.createRandom().connect(env.l1Wallet.provider)
      const receipt = await env.l1Messenger
        .connect(signer)
        .sendMessage(
          tx.contract.address,
          tx.contract.interface.encodeFunctionData(
            tx.functionName,
            tx.functionParams
          ),
          MESSAGE_GAS,
          {
            gasPrice: 0,
          }
        )

      await env.waitForXDomainTransaction(receipt, Direction.L1ToL2)
    })
  )
}

export const executeL2ToL1TransactionsParallel = async (
  env: OptimismEnv,
  txs: TransactionParams[]
) => {
  await Promise.all(
    txs.map(async (tx) => {
      const signer = ethers.Wallet.createRandom().connect(env.l2Wallet.provider)
      const receipt = await env.l2Messenger
        .connect(signer)
        .sendMessage(
          tx.contract.address,
          tx.contract.interface.encodeFunctionData(
            tx.functionName,
            tx.functionParams
          ),
          MESSAGE_GAS,
          {
            gasPrice: 0,
          }
        )

      await env.relayXDomainMessages(receipt)
      await env.waitForXDomainTransaction(receipt, Direction.L2ToL1)
    })
  )
}

export const executeL2TransactionsParallel = async (
  env: OptimismEnv,
  txs: TransactionParams[]
) => {
  await Promise.all(
    txs.map(async (tx) => {
      const signer = ethers.Wallet.createRandom().connect(env.l2Wallet.provider)
      const result = await tx.contract
        .connect(signer)
        .functions[tx.functionName](...tx.functionParams, {
          gasPrice: 0,
        })
      await result.wait()
    })
  )
}

export const executeRepeatedL1ToL2TransactionsParallel = async (
  env: OptimismEnv,
  tx: TransactionParams,
  count: number
) => {
  await executeL1ToL2TransactionsParallel(
    env,
    [...Array(count).keys()].map(() => tx)
  )
}

export const executeRepeatedL2ToL1TransactionsParallel = async (
  env: OptimismEnv,
  tx: TransactionParams,
  count: number
) => {
  await executeL2ToL1TransactionsParallel(
    env,
    [...Array(count).keys()].map(() => tx)
  )
}

export const executeRepeatedL2TransactionsParallel = async (
  env: OptimismEnv,
  tx: TransactionParams,
  count: number
) => {
  await executeL2TransactionsParallel(
    env,
    [...Array(count).keys()].map(() => tx)
  )
}
