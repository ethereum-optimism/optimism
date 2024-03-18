import 'dotenv/config'
import { z } from 'zod'
import metamask from '@synthetixio/synpress/commands/metamask.js'
import synpressPlaywright from '@synthetixio/synpress/commands/playwright.js'
import { confirmPageElements } from '@synthetixio/synpress/pages/metamask/notification-page.js'
import { expect, test, type Page } from '@playwright/test'
import { mnemonicToAccount, privateKeyToAccount } from 'viem/accounts'
import { formatGwei, parseGwei } from 'viem'

import { testWithSynpress } from './testWithSynpressUtil'
import {
  incrementSelfSendTxGauge,
  setFeeEstimationGauge,
} from './prometheusUtils'

const env = z
  .object({
    METAMASK_SECRET_WORDS_OR_PRIVATEKEY: z.string(),
    METAMASK_OP_SEPOLIA_RPC_URL: z.string().url(),
    METAMASK_DAPP_URL: z.string().url(),
  })
  .parse(process.env)

const expectedSender = env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY?.startsWith('0x')
  ? privateKeyToAccount(
      env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY as `0x${string}`
    ).address.toLowerCase()
  : mnemonicToAccount(
      env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY as string
    ).address.toLowerCase()
const expectedRecipient = expectedSender

const expectedCurrencySymbol = 'OPS'

let sharedPage: Page
let wasSuccessful: boolean
let handledFailure: boolean

test.describe.configure({ mode: 'serial' })

test.beforeAll(() => {
  wasSuccessful = false
  handledFailure = false
})

test.afterAll(async () => {
  // This is handling failure scenarios such as Playwright timeouts
  // where are not able to catch and respond to an error.
  if (!wasSuccessful && !handledFailure) {
    await incrementSelfSendTxGauge(false)
  }

  await sharedPage.close()
})

testWithSynpress('Setup wallet and dApp', async ({ page }) => {
  console.log('Setting up wallet and dApp...')
  sharedPage = page
  await sharedPage.goto('http://localhost:9011')
})

testWithSynpress('Add OP Sepolia network', async () => {
  console.log('Adding OP Sepolia network...')
  const expectedChainId = '0xaa37dc'

  await metamask.addNetwork({
    name: 'op-sepolia',
    rpcUrls: {
      default: {
        http: [env.METAMASK_OP_SEPOLIA_RPC_URL],
      },
    },
    id: '11155420',
    nativeCurrency: {
      symbol: expectedCurrencySymbol,
    },
    blockExplorers: {
      default: {
        url: 'https://optimism-sepolia.blockscout.com',
      },
    },
  })

  try {
    await expect(sharedPage.locator('#chainId')).toHaveText(expectedChainId)
  } catch (error) {
    await incrementSelfSendTxGauge(false)
    handledFailure = true
    throw error
  }
})

test(`Connect wallet with ${expectedSender}`, async () => {
  console.log(`Connecting wallet with ${expectedSender}...`)
  await sharedPage.click('#connectButton')
  await metamask.acceptAccess()

  try {
    await expect(sharedPage.locator('#accounts')).toHaveText(expectedSender)
  } catch (error) {
    await incrementSelfSendTxGauge(false)
    handledFailure = true
    throw error
  }
})

test('Send an EIP-1559 transaction and verify success', async () => {
  console.log('Sending an EIP-1559 transaction and verify success...')
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

  const notificationPage =
    await synpressPlaywright.switchToMetamaskNotification()

  console.log('Gathering transaction fee estimations...')
  const lowFeeEstimate = await getFeeEstimateInGwei(
    confirmPageElements.gasOptionLowButton,
    'Low',
    notificationPage
  )

  const highFeeEstimate = await getFeeEstimateInGwei(
    confirmPageElements.gasOptionHighButton,
    'Aggressive',
    notificationPage
  )

  // Medium needs to be last because that's the gas option we want to submit the tx with
  const mediumFeeEstimate = await getFeeEstimateInGwei(
    confirmPageElements.gasOptionMediumButton,
    'Market',
    notificationPage
  )

  console.log('Sent transaction, waiting for confirmation...')
  await metamask.confirmTransactionAndWaitForMining()
  const txHash = await txHashPromise

  const transactionReceiptPromise = new Promise<Record<string, string>>(
    (resolve) => {
      sharedPage.on('load', async () => {
        const responseText = await sharedPage.locator('body > main').innerText()
        const transactionReceipt = JSON.parse(
          responseText.replace('Response: ', '')
        )
        resolve(transactionReceipt)
      })
    }
  )

  // Metamask test dApp allows us access to the Metamask RPC provider via loading this URL.
  // The RPC response will be populated onto the page that's loaded.
  // More info here: https://github.com/MetaMask/test-dapp/tree/main#usage
  console.log('Retrieving transaction receipt...')
  await sharedPage.goto(
    `${env.METAMASK_DAPP_URL}/request.html?method=eth_getTransactionReceipt&params=["${txHash}"]`
  )

  const transactionReceipt = await transactionReceiptPromise

  try {
    expect(transactionReceipt.status).toBe('0x1')
    wasSuccessful = true
    await incrementSelfSendTxGauge(true)
  } catch (error) {
    await incrementSelfSendTxGauge(false)
    handledFailure = true
    throw error
  }

  await setFeeEstimationGauge('low', lowFeeEstimate)
  await setFeeEstimationGauge('medium', mediumFeeEstimate)
  await setFeeEstimationGauge('high', highFeeEstimate)
  await setFeeEstimationGauge('actual', getActualTransactionFee(transactionReceipt))
})

const getFeeEstimateInGwei = async (
  gasOptionButton: string,
  waitForText: 'Low' | 'Market' | 'Aggressive',
  notificationPage: Page
) => {
  await synpressPlaywright.waitAndClick(
    confirmPageElements.editGasFeeButton,
    notificationPage
  )
  await synpressPlaywright.waitAndClick(gasOptionButton, notificationPage)
  await synpressPlaywright.waitForText(
    `${confirmPageElements.editGasFeeButton} .edit-gas-fee-button__label`,
    waitForText,
    notificationPage
  )
  const regexParseEtherValue = /(\d+\.\d+)\s?OPS/
  const feeValue = (
    await synpressPlaywright.waitAndGetValue(
      confirmPageElements.totalLabel,
      notificationPage
    )
  ).match(regexParseEtherValue)[1]
  return parseInt(parseGwei(feeValue).toString())
}

const getActualTransactionFee = (transactionReceipt: Record<string, string>) => {
  const effectiveGasPrice = BigInt(transactionReceipt.effectiveGasPrice)
  const l2GasUsed = BigInt(transactionReceipt.gasUsed)
  const l1Fee = BigInt(transactionReceipt.l1Fee)
  return parseInt(formatGwei(effectiveGasPrice * l2GasUsed + l1Fee, 'wei'))
}
