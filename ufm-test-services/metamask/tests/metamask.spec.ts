import 'dotenv/config'
import metamask from '@synthetixio/synpress/commands/metamask.js'
import { expect, test, type Page } from '@playwright/test'
import { mnemonicToAccount, privateKeyToAccount } from 'viem/accounts'

import { testWithSynpress } from './testWithSynpressUtil'

const expectedSender =
  process.env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY?.startsWith('0x')
    ? privateKeyToAccount(
        process.env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY as `0x${string}`
      ).address.toLowerCase()
    : mnemonicToAccount(
        process.env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY as string
      ).address.toLowerCase()
const expectedRecipient = '0x8fcfbe8953433fd1f2e8375ee99057833e4e1e9e'

let sharedPage: Page

test.describe.configure({ mode: 'serial' })

test.afterAll(async () => {
  await sharedPage.close()
})

testWithSynpress('Setup wallet and dApp', async ({ page }) => {
  sharedPage = page
  await sharedPage.goto('http://localhost:9011')
})

testWithSynpress('Add OP Goerli network', async () => {
  const expectedChainId = '0x1a4'

  await metamask.addNetwork({
    name: 'op-goerli',
    rpcUrls: {
      default: {
        http: [process.env.OP_GOERLI_RPC_URL],
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

  await expect(sharedPage.locator('#chainId')).toHaveText(expectedChainId)
})

test(`Connect wallet with ${expectedSender}`, async () => {
  await sharedPage.click('#connectButton')
  await metamask.acceptAccess()
  await expect(sharedPage.locator('#accounts')).toHaveText(expectedSender)
})

test('Send an EIP-1559 transaciton and verfiy success', async () => {
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

  await metamask.confirmTransaction()
  const txHash = await txHashPromise

    // Waiting for Infura (Metamask given provider) to index our transaction
    await sharedPage.waitForTimeout(10_000)

  // Metamask test dApp allows us access to the Metamask RPC provider via loading this URL.
  // The RPC reponse will be populated onto the page that's loaded.
  // More info here: https://github.com/MetaMask/test-dapp/tree/main#usage
  await sharedPage.goto(
    `${process.env.METAMASK_DAPP_URL}/request.html?method=eth_getTransactionReceipt&params=["${txHash}"]`
  )

  // Waiting for RPC response to be populated on the page
  await sharedPage.waitForTimeout(2_000)

  const transaction = JSON.parse(
    (await sharedPage.locator('body > main').innerText()).replace(
      'Response: ',
      ''
    )
  )
  expect(transaction.status).toBe('0x1')
})
