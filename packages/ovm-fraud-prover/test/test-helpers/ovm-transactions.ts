import { Contract, Wallet, BigNumber } from "ethers";
import { TransactionRequest } from "@ethersproject/providers"
import * as rlp from 'rlp'

import { OVMTransactionData } from "../../src/interfaces";
import { NULL_ADDRESS, GAS_LIMIT } from "./constants";
import { toHexString } from "./buffer-utils"

export const makeOvmTransaction = (
  contract: Contract,
  wallet: Wallet,
  functionName: string,
  functionParams: any[] = []
): OVMTransactionData => {
  return {
    timestamp: Math.floor(Date.now() / 1000),
    queueOrigin: 0,
    ovmEntrypoint: contract.address,
    callBytes: contract.interface.encodeFunctionData(functionName, functionParams),
    fromAddress: wallet.address,
    l1MsgSenderAddress: NULL_ADDRESS,
    allowRevert: false,
  }
}

export const signAndSendOvmTransaction = async (
  wallet: Wallet,
  transaction: OVMTransactionData
): Promise<void> => {
  const transactionRequest: TransactionRequest = {
    to: transaction.ovmEntrypoint,
    from: transaction.fromAddress,
    nonce: await wallet.getTransactionCount(),
    gasLimit: GAS_LIMIT,
    gasPrice: 0,
    data: transaction.callBytes,
    value: BigNumber.from(0),
    chainId: 0
  }

  const signedTransaction = await wallet.signTransaction(transactionRequest)

  await wallet.provider.sendTransaction(signedTransaction)
}

export const encodeTransaction = (transaction: OVMTransactionData): string => {
  return toHexString(
    rlp.encode([
      transaction.timestamp,
      transaction.queueOrigin,
      transaction.ovmEntrypoint,
      transaction.callBytes,
      transaction.fromAddress,
      transaction.l1MsgSenderAddress,
      transaction.allowRevert ? 1 : 0,
    ])
  )
}