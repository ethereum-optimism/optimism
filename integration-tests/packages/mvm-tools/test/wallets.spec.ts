import { expect } from 'chai'
import assert = require('assert')
import ethers = require('ethers')


import {
  Contract, ContractFactory, Wallet,
} from 'ethers'

describe('Wallet tools in test', async () => {
  before(async () => {
  })

  it.skip('create random wallet private key tool', async () => {
    var pk=Wallet.createRandom().privateKey;
    console.log(pk);
  })
})