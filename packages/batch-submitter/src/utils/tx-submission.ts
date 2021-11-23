import { Signer, ethers, PopulatedTransaction } from 'ethers'
import {
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import * as ynatm from '@eth-optimism/ynatm'

export interface ResubmissionConfig {
  resubmissionTimeout: number
  minGasPriceInGwei: number
  maxGasPriceInGwei: number
  gasRetryIncrement: number
}

export type SubmitTransactionFn = (
  tx: PopulatedTransaction
) => Promise<TransactionReceipt>

export interface TxSubmissionHooks {
  beforeSendTransaction: (tx: PopulatedTransaction) => void
  onTransactionResponse: (txResponse: TransactionResponse) => void
}

const getGasPriceInGwei = async (signer: Signer): Promise<number> => {
  return parseInt(
    ethers.utils.formatUnits(await signer.getGasPrice(), 'gwei'),
    10
  )
}

export const ynatmRejectOn = (e) => {
  // taken almost verbatim from the readme,
  // see https://github.com/ethereum-optimism/ynatm.
  // immediately rejects on reverts and nonce errors
  const errMsg = e.toString().toLowerCase()
  const conditions = ['revert', 'nonce']
  for (const cond of conditions) {
    if (errMsg.includes(cond)) {
      return true
    }
  }

  return false
}

export const submitTransactionWithYNATM = async (
  tx: PopulatedTransaction,
  signer: Signer,
  config: ResubmissionConfig,
  numConfirmations: number,
  hooks: TxSubmissionHooks
): Promise<TransactionReceipt> => {
  const sendTxAndWaitForReceipt = async (
    gasPrice
  ): Promise<TransactionReceipt> => {
    const fullTx = {
      ...tx,
      gasPrice,
    }
    hooks.beforeSendTransaction(fullTx)
    const txResponse = await signer.sendTransaction(fullTx)
    hooks.onTransactionResponse(txResponse)
    return signer.provider.waitForTransaction(txResponse.hash, numConfirmations)
  }

  const minGasPrice = await getGasPriceInGwei(signer)
  const receipt = await ynatm.send({
    sendTransactionFunction: sendTxAndWaitForReceipt,
    minGasPrice: ynatm.toGwei(minGasPrice),
    maxGasPrice: ynatm.toGwei(config.maxGasPriceInGwei),
    gasPriceScalingFunction: ynatm.LINEAR(config.gasRetryIncrement),
    delay: config.resubmissionTimeout,
    rejectImmediatelyOnCondition: ynatmRejectOn,
  })
  return receipt
}

export interface TransactionSubmitter {
  submitTransaction(
    tx: PopulatedTransaction,
    hooks?: TxSubmissionHooks
  ): Promise<TransactionReceipt>
}

export class YnatmTransactionSubmitter implements TransactionSubmitter {
  constructor(
    readonly signer: Signer,
    readonly ynatmConfig: ResubmissionConfig,
    readonly numConfirmations: number
  ) {}

  public async submitTransaction(
    tx: PopulatedTransaction,
    hooks?: TxSubmissionHooks
  ): Promise<TransactionReceipt> {
    if (!hooks) {
      hooks = {
        beforeSendTransaction: () => undefined,
        onTransactionResponse: () => undefined,
      }
    }
    return submitTransactionWithYNATM(
      tx,
      this.signer,
      this.ynatmConfig,
      this.numConfirmations,
      hooks
    )
  }
}
