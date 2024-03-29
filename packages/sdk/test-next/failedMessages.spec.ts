import { describe, it, expect } from 'vitest'
import { Address, Hex, encodePacked, keccak256, toHex } from 'viem'
import { ethers } from 'ethers'
import { z } from 'zod'
import { hashCrossDomainMessagev1 } from '@eth-optimism/core-utils'
import { optimismSepolia } from 'viem/chains'

import { CONTRACT_ADDRESSES, CrossChainMessenger } from '../src'
import { sepoliaPublicClient, sepoliaTestClient } from './testUtils/viemClients'
import { sepoliaProvider, opSepoliaProvider } from './testUtils/ethersProviders'

/**
 * Generated on Mar 28 2024 using
 * `forge inspect L1CrossDomainMessenger storage-layout`
 **/
const failedMessagesStorageLayout = {
  astId: 7989,
  contract: 'src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger',
  label: 'failedMessages',
  offset: 0,
  slot: 206n,
  type: 't_mapping(t_bytes32,t_bool)',
}

const sepoliaCrossDomainMessengerAddress = CONTRACT_ADDRESSES[
  optimismSepolia.id
].l1.L1CrossDomainMessenger as Address

const setMessageAsFailed = async (tx: Hex) => {
  const message = await crossChainMessenger.toCrossChainMessage(tx)
  const messageHash = hashCrossDomainMessagev1(
    message.messageNonce,
    message.sender,
    message.target,
    message.value,
    message.minGasLimit,
    message.message
  ) as Hex

  const keySlotHash = keccak256(
    encodePacked(
      ['bytes32', 'uint256'],
      [messageHash, failedMessagesStorageLayout.slot]
    )
  )
  return sepoliaTestClient.setStorageAt({
    address: sepoliaCrossDomainMessengerAddress,
    index: keySlotHash,
    value: toHex(true, { size: 32 }),
  })
}

const E2E_PRIVATE_KEY = z
  .string()
  .describe('Private key')
  // Mnemonic:          test test test test test test test test test test test junk
  .default('0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6')
  .parse(import.meta.env.VITE_E2E_PRIVATE_KEY)

const sepoliaWallet = new ethers.Wallet(E2E_PRIVATE_KEY, sepoliaProvider)
const crossChainMessenger = new CrossChainMessenger({
  l1SignerOrProvider: sepoliaWallet,
  l2SignerOrProvider: opSepoliaProvider,
  l1ChainId: 11155111,
  l2ChainId: 11155420,
  bedrock: true,
})

describe('replaying failed messages', () => {
  it('should be able to replay failed messages', async () => {
    // Grab an existing tx but mark it as failed
    // @see https://sepolia-optimism.etherscan.io/tx/0x28249a36f764afab583a4633d59ff6c2a0e934293062bffa7cedb662e5da9abd
    const tx =
      '0x28249a36f764afab583a4633d59ff6c2a0e934293062bffa7cedb662e5da9abd'

    await setMessageAsFailed(tx)

    // debugging ethers.js is brutal because of error message so let's instead
    // send the tx with viem. If it succeeds we will then test with ethers
    const txData =
      await crossChainMessenger.populateTransaction.finalizeMessage(tx)

    await sepoliaPublicClient.call({
      data: txData.data as Hex,
      to: txData.to as Address,
    })

    // finalize the message
    const finalizeTx = await crossChainMessenger.finalizeMessage(tx)

    const receipt = await finalizeTx.wait()

    expect(receipt.transactionHash).toBeDefined()
  })
})
