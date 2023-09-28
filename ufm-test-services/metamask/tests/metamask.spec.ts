import 'dotenv/config'
import { z } from 'zod'
import metamask from '@synthetixio/synpress/commands/metamask.js'
import { expect, test, type Page } from '@playwright/test'
import { mnemonicToAccount, privateKeyToAccount } from 'viem/accounts'

import { testWithSynpress } from './testWithSynpressUtil'
import {
  incrementMetamaskTxCounter,
  setMetamaskTxCounter,
} from './prometheusUtils'

const env = z.object({
  METAMASK_SECRET_WORDS_OR_PRIVATEKEY: z.string(),
  METAMASK_OP_GOERLI_RPC_URL: z.string().url(),
  METAMASK_DAPP_URL: z.string().url()
}).parse(process.env)

const expectedSender =
  env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY?.startsWith('0x')
    ? privateKeyToAccount(
        env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY as `0x${string}`
      ).address.toLowerCase()
    : mnemonicToAccount(
        env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY as string
      ).address.toLowerCase()
const expectedRecipient = '0x8fcfbe8953433fd1f2e8375ee99057833e4e1e9e'

let sharedPage: Page

test.describe.configure({ mode: 'serial' })

test.afterAll(async () => {
  await sharedPage.close()
})

testWithSynpress('Setup wallet and dApp', async ({ page }) => {
  console.log('Seting up wallet and dApp...')
  sharedPage = page
  await sharedPage.goto('http://localhost:9011')
  console.log('Setup wallet and dApp')
})

testWithSynpress('Add OP Goerli network', async () => {
  console.log('Adding OP Goerli network...')
  const expectedChainId = '0x1a4'

  await metamask.addNetwork({
    name: 'op-goerli',
    rpcUrls: {
      default: {
        http: [env.METAMASK_OP_GOERLI_RPC_URL],
      },
    },
    id: '420',
    nativeCurrency: {
      symbol: 'OPG',
    },
    blockExplorers: {
      default: {
        url: 'https://goerli-explorer.optimism.io',
      },
    },
  })

  try {
    await expect(sharedPage.locator('#chainId')).toHaveText(expectedChainId)
  } catch (error) {
    await setMetamaskTxCounter(true, 0)
    await incrementMetamaskTxCounter(false)
    throw error
  }
  console.log('Added OP Goerli network')
})

test(`Connect wallet with ${expectedSender}`, async () => {
  console.log(`Connecting wallet with ${expectedSender}...`)
  await sharedPage.click('#connectButton')
  await metamask.acceptAccess()

  try {
    await expect(sharedPage.locator('#accounts')).toHaveText(expectedSender)
  } catch (error) {
    await setMetamaskTxCounter(true, 0)
    await incrementMetamaskTxCounter(false)
    throw error
  }
  console.log(`Connected wallet with ${expectedSender}`)
})

test('Send an EIP-1559 transaciton and verfiy success', async () => {
  console.log('Sending an EIP-1559 transaciton and verfiy success...')
  const expectedTransferAmount = '0x1'
  const expectedTxType = '0x2'

  await sharedPage.locator('#toInput').fill(expectedRecipient)
  await sharedPage.locator('#amountInput').fill(expectedTransferAmount)
  await sharedPage.locator('#typeInput').selectOption(expectedTxType)

  await sharedPage.click('#submitForm')

  const txHashPromise = new Promise((resolve) => {
    // Metamask test dApp only console.logs the transaction hash,
    // so we must setup a listener before we confirm the tx to capture it
    sharedPage.on('console', async (msg) => {
      resolve(msg.text()) // Resolve the Promise when txHash is set
    })
  })

  await metamask.confirmTransactionAndWaitForMining()
  const txHash = await txHashPromise

  // Metamask test dApp allows us access to the Metamask RPC provider via loading this URL.
  // The RPC reponse will be populated onto the page that's loaded.
  // More info here: https://github.com/MetaMask/test-dapp/tree/main#usage
  await sharedPage.goto(
    `${env.METAMASK_DAPP_URL}/request.html?method=eth_getTransactionReceipt&params=["${txHash}"]`
  )

  // Waiting for RPC response to be populated on the page
  await sharedPage.waitForTimeout(2_000)

  const transaction = JSON.parse(
    (await sharedPage.locator('body > main').innerText()).replace(
      'Response: ',
      ''
    )
  )

  try {
    expect(transaction.status).toBe('0x1')
    await setMetamaskTxCounter(false, 0)
    await incrementMetamaskTxCounter(true)
  } catch (error) {
    await setMetamaskTxCounter(true, 0)
    await incrementMetamaskTxCounter(false)
    throw error
  }
  console.log('Sent an EIP-1559 transaciton and verfied success')
})
