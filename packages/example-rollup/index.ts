/* Imports */
import { BaseDB, SimpleClient } from '@pigi/core'
import MemDown from 'memdown'
import {
  State,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  UnipigWallet,
  FaucetRequest,
  SignedTransactionReceipt,
} from '@pigi/wallet'
import { ethers } from 'ethers'

/* Global declarations */
declare var document: any

const UNISWAP_ADDRESS = '0x' + 'ff'.repeat(32)

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

const wallet: ethers.Wallet = ethers.Wallet.createRandom()

// Connect to the mock aggregator
unipigWallet.rollup.connect(new SimpleClient('http://localhost:3000'))

updateAccountAddress(wallet.address)

async function fetchBalanceUpdate() {
  const balances = await unipigWallet.getBalances(wallet.address)
  const uniswapBalances = await unipigWallet.rollup.getUniswapBalances()
  updateBalances(balances)
  updateUniswapBalances(uniswapBalances)
}

async function onRequestFundsClicked() {
  const transaction: FaucetRequest = {
    requester: wallet.address,
    amount: 10,
  }
  const response = await unipigWallet.rollup.requestFaucetFunds(
    transaction,
    wallet.address
  )
  updateBalances(response)
}

async function onTransferFundsClicked() {
  const selectedIndex = document.getElementById('send-token-type').selectedIndex
  const tokenType = selectedIndex === 0 ? UNI_TOKEN_TYPE : PIGI_TOKEN_TYPE
  const amount = parseInt(document.getElementById('send-amount').value, 10)
  const recipient = document.getElementById('send-recipient').value
  const response: SignedTransactionReceipt = await unipigWallet.rollup.sendTransaction(
    {
      tokenType,
      recipient,
      amount,
    },
    wallet.address
  )
  updateBalances(
    response.transactionReceipt.updatedState[wallet.address].balances
  )
}

async function onSwapFundsClicked() {
  const selectedIndex = document.getElementById('swap-token-type').selectedIndex
  const tokenType = selectedIndex === 0 ? UNI_TOKEN_TYPE : PIGI_TOKEN_TYPE
  const inputAmount = parseInt(document.getElementById('swap-amount').value, 10)
  const response: SignedTransactionReceipt = await unipigWallet.rollup.sendTransaction(
    {
      tokenType,
      inputAmount,
      minOutputAmount: 0,
      timeout: +new Date() + 1000,
    },
    wallet.address
  )
  updateBalances(
    response.transactionReceipt.updatedState[wallet.address].balances
  )
  updateUniswapBalances(
    response.transactionReceipt.updatedState[UNISWAP_ADDRESS].balances
  )
}

fetchBalanceUpdate()
