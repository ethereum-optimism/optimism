#!/usr/bin/env ts-node

/**
 * Utility that will relay all L2 => L1 messages created within a given L2 transaction.
 */

/* Imports: External */
import { ethers } from 'ethers'
import { predeploys, getContractInterface } from '@eth-optimism/contracts'
import { sleep } from '@eth-optimism/core-utils'
import dotenv from 'dotenv'

/* Imports: Internal */
import { getMessagesAndProofsForL2Transaction } from '../relay-tx'

dotenv.config()
const l1RpcProviderUrl = process.env.WITHDRAW__L1_RPC_URL
const l2RpcProviderUrl = process.env.WITHDRAW__L2_RPC_URL
const l1PrivateKey = process.env.WITHDRAW__L1_PRIVATE_KEY
const l1StateCommitmentChainAddress =
  process.env.WITHDRAW__STATE_COMMITMENT_CHAIN_ADDRESS
const l1CrossDomainMessengerAddress =
  process.env.WITHDRAW__L1_CROSS_DOMAIN_MESSENGER_ADDRESS

const main = async () => {
  const l2TransactionHash = process.argv[2]
  if (l2TransactionHash === undefined) {
    throw new Error(`must provide l2 transaction hash`)
  }

  const l1RpcProvider = new ethers.providers.JsonRpcProvider(l1RpcProviderUrl)
  const l1Wallet = new ethers.Wallet(l1PrivateKey, l1RpcProvider)
  const l1WalletBalance = await l1Wallet.getBalance()
  console.log(`relayer address: ${l1Wallet.address}`)
  console.log(`relayer balance: ${ethers.utils.formatEther(l1WalletBalance)}`)

  const l1CrossDomainMessenger = new ethers.Contract(
    l1CrossDomainMessengerAddress,
    getContractInterface('OVM_L1CrossDomainMessenger'),
    l1Wallet
  )

  console.log(`searching for messages in transaction: ${l2TransactionHash}`)
  let messagePairs = []
  while (true) {
    try {
      messagePairs = await getMessagesAndProofsForL2Transaction(
        l1RpcProviderUrl,
        l2RpcProviderUrl,
        l1StateCommitmentChainAddress,
        predeploys.OVM_L2CrossDomainMessenger,
        l2TransactionHash
      )
      break
    } catch (err) {
      if (err.message.includes('unable to find state root batch for tx')) {
        console.log(`no state root batch for tx yet, trying again in 5s...`)
        await sleep(5000)
      } else {
        throw err
      }
    }
  }

  console.log(`found ${messagePairs.length} messages`)
  for (let i = 0; i < messagePairs.length; i++) {
    console.log(`relaying message ${i + 1}/${messagePairs.length}`)
    const { message, proof } = messagePairs[i]
    while (true) {
      try {
        const result = await l1CrossDomainMessenger.relayMessage(
          message.target,
          message.sender,
          message.message,
          message.messageNonce,
          proof
        )
        await result.wait()
        console.log(
          `relayed message ${i + 1}/${messagePairs.length}! L1 tx hash: ${
            result.hash
          }`
        )
        break
      } catch (err) {
        if (err.message.includes('execution failed due to an exception')) {
          console.log(`fraud proof may not be elapsed, trying again in 5s...`)
          await sleep(5000)
        } else if (err.message.includes('message has already been received')) {
          console.log(
            `message ${i + 1}/${
              messagePairs.length
            } was relayed by someone else`
          )
          break
        } else {
          throw err
        }
      }
    }
  }
}

main()
