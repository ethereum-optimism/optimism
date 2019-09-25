/* Imports */
import {
  DB,
  newInMemoryDB,
  SignedByDB,
  SignedByDecider,
  SimpleClient,
  getLogger,
} from '@pigi/core'
import {
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  UNISWAP_ADDRESS,
  UnipigTransitioner,
  RollupClient,
  Balances,
  RollupStateSolver,
  DefaultRollupStateSolver,
} from '@pigi/wallet'
import { ethers } from 'ethers'

const log = getLogger('simple-client')

/* Global declarations */
declare var document: any

/* Functions which update UI */
const updateAccountAddress = (address) => {
  document.getElementById('account-address').textContent = address
}

const updateBalances = (balances) => {
  if (typeof balances === 'undefined') {
    log.debug('Undefined balances!')
    return
  }
  document.getElementById('uni-balance').textContent = balances[UNI_TOKEN_TYPE]
  document.getElementById('pigi-balance').textContent =
    balances[PIGI_TOKEN_TYPE]
}

const updateUniswapBalances = (balances) => {
  document.getElementById('uniswap-uni-balance').textContent =
    balances[UNI_TOKEN_TYPE]
  document.getElementById('uniswap-pigi-balance').textContent =
    balances[PIGI_TOKEN_TYPE]
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
let unipigWallet
let wallet: ethers.Wallet

async function initialize() {
  wallet = ethers.Wallet.createRandom()

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
  unipigWallet = new UnipigTransitioner(
    newInMemoryDB(),
    rollupStateSolver,
    rollupClient,
    undefined,
    undefined,
    wallet
  )
  // Update account address
  updateAccountAddress(wallet.address)
  // Connect to the mock aggregator
  rollupClient.connect(new SimpleClient('http://localhost:3000'))
  await fetchBalanceUpdate()
}

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

initialize()
