/* Imports */
import {
  DB,
  newInMemoryDB,
  SignedByDB,
  SignedByDecider,
  SimpleClient,
} from '@pigi/core'
import {
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  UnipigTransitioner,
  RollupClient,
  Balances,
  RollupStateSolver,
  DefaultRollupStateSolver,
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

const wallet: ethers.Wallet = ethers.Wallet.createRandom()

const signatureDB: DB = newInMemoryDB()
const signedByDB: SignedByDB = new SignedByDB(signatureDB)
const signedByDecider: SignedByDecider = new SignedByDecider(
  signedByDB,
  Buffer.from(wallet.address)
)
const rollupStateSolver: RollupStateSolver = new DefaultRollupStateSolver(
  signedByDB,
  signedByDecider
)
const rollupClient: RollupClient = new RollupClient(newInMemoryDB())
const unipigWallet = new UnipigTransitioner(
  newInMemoryDB(),
  rollupStateSolver,
  rollupClient
)
// Now create a wallet account

// Connect to the mock aggregator
rollupClient.connect(new SimpleClient('http://localhost:3000'))

updateAccountAddress(wallet.address)

async function fetchBalanceUpdate() {
  const balances = await unipigWallet.getBalances(wallet.address)
  const uniswapBalances = await unipigWallet.getUniswapBalances()
  updateBalances(balances)
  updateUniswapBalances(uniswapBalances)
}

async function onRequestFundsClicked() {
  await unipigWallet.requestFaucetFunds(wallet.address, 10)
  const updatedBalances: Balances = await unipigWallet.getBalances(
    wallet.address
  )
  updateBalances(updatedBalances)
}

async function onTransferFundsClicked() {
  const selectedIndex = document.getElementById('send-token-type').selectedIndex
  const tokenType = selectedIndex === 0 ? UNI_TOKEN_TYPE : PIGI_TOKEN_TYPE
  const amount = parseInt(document.getElementById('send-amount').value, 10)
  const recipient = document.getElementById('send-recipient').value

  await unipigWallet.send(tokenType, wallet.address, recipient, amount)
  const updatedBalances: Balances = await unipigWallet.getBalances(
    wallet.address
  )

  updateBalances(updatedBalances)
}

async function onSwapFundsClicked() {
  const selectedIndex = document.getElementById('swap-token-type').selectedIndex
  const tokenType = selectedIndex === 0 ? UNI_TOKEN_TYPE : PIGI_TOKEN_TYPE
  const inputAmount = parseInt(document.getElementById('swap-amount').value, 10)
  await unipigWallet.swap(
    tokenType,
    wallet.address,
    inputAmount,
    0,
    +new Date() + 1000
  )
  const [senderBalance, uniswapBalance] = await Promise.all([
    unipigWallet.getBalances(wallet.address),
    unipigWallet.getBalances(UNISWAP_ADDRESS),
  ])
  updateBalances(senderBalance)
  updateUniswapBalances(uniswapBalance)
}

fetchBalanceUpdate()
