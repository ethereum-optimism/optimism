/* External Imports */
import { Contract, Wallet } from "ethers";
import * as rlp from 'rlp'

/* Internal Imports */
import { OVMTransactionData } from "../../src/interfaces";
import { toHexString } from "../../src/utils"
import { NULL_ADDRESS, GAS_LIMIT } from "./constants";

/**
 * Generates an OVM transaction.
 * @param contract Contract to send the transaction to.
 * @param wallet Ethers wallet to send the transaction from.
 * @param functionName Name of the function to call.
 * @param functionParams Parameters to the function call.
 * @returns An OVM transaction data object.
 */
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

/**
 * Sends an OVM transaction.
 * @param wallet Ethers wallet to send the transaction from.
 * @param transaction Transaction data to send.
 */
export const signAndSendOvmTransaction = async (
  wallet: Wallet,
  transaction: OVMTransactionData
): Promise<void> => {
  await wallet.sendTransaction({
    to: transaction.ovmEntrypoint,
    from: transaction.fromAddress,
    gasLimit: GAS_LIMIT,
    data: transaction.callBytes
  })
}

/**
 * Encodes an OVM transaction.
 * @param transaction OVM transaction to encode.
 * @returns Encoded transaction.
 */
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