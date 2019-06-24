/* External Imports */
import {
    abi,
    AbiStateObject,
    AbiRange,
    hexStringify,
    AbiOwnershipParameters,
    AbiOwnershipTransaction,
  } from '@pigi/core'
  import BigNum = require('bn.js')
  /* Contract Imports */
  import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
  import * as BasicTokenMock from '../build/BasicTokenMock.json'
  import * as Deposit from '../build/Deposit.json'
  import * as Commitment from '../build/CommitmentChain.json'
  import * as TransactionPredicate from '../build/TransactionPredicate.json'
  import * as OwnershipTransactionPredicate from '../build/OwnershipTransactionPredicate.json'
  /* Logging */
  import debug from 'debug'
  import { check } from 'ethers/utils/wordlist'
  const log = debug('test:info:state-ownership')
  /* Testing Setup */
  import chai = require('chai')
  export const should = chai.should()
  
  describe('Deposit with Ownership', () => {
    const provider = createMockProvider()
    const [wallet, walletTo] = getWallets(provider)
    let depositContract
    let commitmentContract
    let ownershipPredicate
  
    beforeEach(async () => {
      commitmentContract = await deployContract(wallet, Commitment, [])
  
    describe('Commitment Contract', () => {
      it('does not throw when deposit is called after approving erc20 movement', async () => {

      })
  })
  