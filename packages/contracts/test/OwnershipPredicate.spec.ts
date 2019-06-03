/* External Imports */
import { abi, AbiRange, AbiStateObject, AbiStateUpdate, hexStringify } from '@pigi/utils'
import BigNum = require('bn.js')

/* Logging */
import debug from 'debug'
const log = debug('test:info:ownership-predicate')

import chai = require('chai')
import {createMockProvider, deployContract, getWallets, solidity} from 'ethereum-waffle';
import * as OwnershipPredicate from '../build/OwnershipPredicate.json'
import * as BasicTokenMock from '../build/BasicTokenMock.json'
import * as Deposit from '../build/Deposit.json'

chai.use(solidity);
const {expect} = chai;

describe.only('OwnershipPredicate', () => {
  const provider = createMockProvider()
  const [wallet, walletTo] = getWallets(provider)
  let ownershipPredicate
  let token
  let depositContract

  beforeEach(async () => {
    token = await deployContract(wallet, BasicTokenMock, [wallet.address, 1000])
    depositContract = await deployContract(wallet, Deposit, [token.address])
    ownershipPredicate = await deployContract(wallet, OwnershipPredicate, [depositContract.address])
  })

  it('should allow exits to be started on deposits', async () => {
    // Deposit some money into the ownership predicate
    await token.approve(depositContract.address, 500)
    const depositData = abi.encode(['address'], [wallet.address])
    const depositStateObject = new AbiStateObject(ownershipPredicate.address, depositData)
    await depositContract.deposit(100, depositStateObject)
    // Attempt to start an exit on this deposit
    const depositRange = { start: hexStringify(new BigNum(0)), end: hexStringify(new BigNum(100)) }
    await ownershipPredicate.startExit({
      stateUpdate: {
        range: depositRange,
        stateObject: depositStateObject,
        depositAddress: depositContract.address,
        plasmaBlockNumber: 0
      },
      subrange: depositRange
    })
  })
});
