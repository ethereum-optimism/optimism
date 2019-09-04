/* Imports */
import { BaseDB, SimpleClient } from '@pigi/core'
import MemDown from 'memdown'
import {
  State,
  UNISWAP_ADDRESS,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  AGGREGATOR_ADDRESS,
  UnipigWallet,
} from '@pigi/wallet'

/* Global declarations */
declare var document: any

/* Functions which update UI */
const updateAccountAddress = (address) => {
  document.getElementById('account-address').textContent = address
}

const updateBalances = (balances) => {
  document.getElementById('uni-balance').textContent = balances.uni
  document.getElementById('pigi-balance').textContent = balances.pigi
}

const updateUniswapBalances = (balances) => {
  document.getElementById('uniswap-uni-balance').textContent = balances.uni
  document.getElementById('uniswap-pigi-balance').textContent = balances.pigi
}

/* Listeners */
setTimeout(() => {
  // Add listener for request funds
  document
    .getElementById('request-funds-button')
    .addEventListener('click', onRequestFundsClicked)
  // Add listener for transfer
  document
    .getElementById('send-button')
    .addEventListener('click', onTransferFundsClicked)
  // Add listener for swap
  document
    .getElementById('swap-button')
    .addEventListener('click', onSwapFundsClicked)
}, 300)

/*
 * Body
 */
const db = new BaseDB(new MemDown('ovm') as any)
const unipigWallet = new UnipigWallet(db)
// Now create a wallet account
const accountAddress = 'mocked account'

// Connect to the mock aggregator
unipigWallet.rollup.connect(new SimpleClient('http://localhost:3000'))

updateAccountAddress(accountAddress)

async function fetchBalanceUpdate() {
  const balances = await unipigWallet.getBalances(accountAddress)
  const uniswapBalances = await unipigWallet.rollup.getUniswapBalances()
  updateBalances(balances)
  updateUniswapBalances(uniswapBalances)
}

async function onRequestFundsClicked() {
  const response = await unipigWallet.rollup.requestFaucetFunds(
    accountAddress,
    10
  )
  updateBalances(response)
}

async function onTransferFundsClicked() {
  const selectedIndex = document.getElementById('send-token-type').selectedIndex
  const tokenType = selectedIndex === 0 ? UNI_TOKEN_TYPE : PIGI_TOKEN_TYPE
  const amount = parseInt(document.getElementById('send-amount').value, 10)
  const recipient = document.getElementById('send-recipient').value
  const response: State = await unipigWallet.rollup.sendTransaction(
    {
      tokenType,
      recipient,
      amount,
    },
    accountAddress
  )
  updateBalances(response.sender.balances)
}

async function onSwapFundsClicked() {
  const selectedIndex = document.getElementById('swap-token-type').selectedIndex
  const tokenType = selectedIndex === 0 ? UNI_TOKEN_TYPE : PIGI_TOKEN_TYPE
  const inputAmount = parseInt(document.getElementById('swap-amount').value, 10)
  const response: State = await unipigWallet.rollup.sendTransaction(
    {
      tokenType,
      inputAmount,
      minOutputAmount: 0,
      timeout: +new Date() + 1000,
    },
    accountAddress
  )
  updateBalances(response.sender.balances)
  updateUniswapBalances(response.uniswap.balances)
}

fetchBalanceUpdate()
