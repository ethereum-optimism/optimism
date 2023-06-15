import { ethers } from 'ethers'
import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import { z } from 'zod'

import { CrossChainMessenger } from '../src'
import { anvilTestUtilFactory, AnvilTestUtil } from './testUtils/anvil'

const { VITE_E2E_RPC_URL_L1_MAINNET, VITE_E2E_RPC_URL_L2_MAINNET } = z
  .object({
    VITE_E2E_RPC_URL_L1_MAINNET: z.string().url(),
    VITE_E2E_RPC_URL_L2_MAINNET: z.string().url(),
  })
  .parse({
    VITE_E2E_RPC_URL_L1_MAINNET: process.env.VITE_E2E_RPC_URL_L1_MAINNET!,
    VITE_E2E_RPC_URL_L2_MAINNET: process.env.VITE_E2E_RPC_URL_L2_MAINNET!,
  })

// TODO maybe move this to test utils
// L2 withdrawal example https://optimistic.etherscan.io/tx/0xd73f0cdf499830f2919f3009cee35611abf5a01842c4eed0e8c2493a00969a5b
describe('Multiple withdrawals in one tx', () => {
  let testUtil: AnvilTestUtil
  beforeAll(async () => {
    type TODO = any
    testUtil =
      await anvilTestUtilFactory({
        l1: {
          forkUrl: VITE_E2E_RPC_URL_L1_MAINNET,
          // forkBlockNumber: TODO,
        },
        l2: {
          forkUrl: VITE_E2E_RPC_URL_L2_MAINNET,
          // forkBlockNumber: TODO,
        },
      } as TODO)
    console.log('Starting anvill1...')
    await testUtil.anvilL1.start()
    console.log('Starting anvill2...')
    await testUtil.anvilL2.start()
    console.log('started')
  })

  afterAll(async () => {
    await testUtil.anvilL1.stop()
    await testUtil.anvilL2.stop()
  })
  it('should be able to do multiple withdrawals in a single batch and then prove and claim them on l1', async () => {
    console.log(testUtil)
    const messenger = new CrossChainMessenger({
      l1ChainId: 1,
      l2ChainId: 10,
      l1SignerOrProvider: new ethers.Wallet(testUtil.anvilAccounts[0], new ethers.providers.JsonRpcProvider(VITE_E2E_RPC_URL_L1_MAINNET)),
      l2SignerOrProvider: new ethers.Wallet(testUtil.anvilAccounts[0], new ethers.providers.JsonRpcProvider(VITE_E2E_RPC_URL_L2_MAINNET)),
    })

    const txHash =
      '0xd73f0cdf499830f2919f3009cee35611abf5a01842c4eed0e8c2493a00969a5b'
    const txReceipt = await testUtil.publicClientL2.getTransactionReceipt({
      hash: txHash,
    })

    expect(txReceipt).toBeDefined()

    const tx = await messenger.proveMessage(txHash)
    const receipt = await tx.wait()

    // A 1 means the transaction was successful
    expect(receipt.status).toBe(1)
    await expect(
      messenger.getProvenWithdrawal(txHash)
    ).resolves.toMatchInlineSnapshot()
  })
})

