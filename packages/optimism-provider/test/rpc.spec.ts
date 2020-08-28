/**
 * Optimism Copyright 2020.
 */

import { SimpleServer, isHexString } from '@eth-optimism/core-utils'
import { OptimismProvider } from '../src/index'
import { ContractFactory } from 'ethers';
import chai = require('chai')
import assert = require('assert')
import chaiAsPromised = require('chai-as-promised')
import { ganache } from '@eth-optimism/ovm-toolchain'

// debugging
import { JsonRpcProvider } from '@ethersproject/providers'

chai.use(chaiAsPromised)
const should = chai.should()

describe('RPC', () => {
  const server = ganache.server({})
  let provider

  const addr = '0x8fd00f170fdf3772c5ebdcd90bf257316c69ba45'
  const contract = '0xdac17f958d2ee523a2206206994597c13d831ec7'

  // Set up the provider and the RPC server
  before(async () => {
    provider = new OptimismProvider('http://localhost:3000')
    //provider = new OptimismProvider('http://192.168.1.112:8545')
    //provider = new JsonRpcProvider('http://192.168.1.112:8545')
    await server.listen(3000)
  })

  after(async () => {
    await server.close()
  })

  it('should send a rpc request', async () => {
    const res = await provider.send('eth_blockNumber', []);
    res.should.be.a('string')
    assert(isHexString(res))
  })

  // TODO(mark): subject to change
  it('should getBlockNumber', async () => {
    const res = await provider.getBlockNumber()
    res.should.be.a('number')
  })

  it('should getGasPrice', async () => {
    const res = await provider.getGasPrice()
    // should be a BigNumber with `_isBigNumber` set to true
    assert(res)
    assert(res._isBigNumber)
  })

  it('should get balance', async () => {
    const res = await provider.getBalance(addr)
    assert(res)
    assert(res._isBigNumber)
  })

  it('should getTransactionCount', async () => {
    const res = await provider.getTransactionCount(addr)
    res.should.be.a('number')
  })

  it('should getCode', async () => {
    const res = await provider.getCode(contract)
    res.should.be.a('string')
    assert(isHexString(res))
  })

  // TODO(mark): subject to change
  it('should getBlock', async () => {
    const res = await provider.getBlock(0)
    res.should.be.a('object')

    res.hash.should.be.a('string')
    res.parentHash.should.be.a('string')
    res.number.should.be.a('number')
    res.timestamp.should.be.a('number')
    res.nonce.should.be.a('string')
    res.difficulty.should.be.a('number')
    assert(res.gasLimit._isBigNumber)
    assert(res.gasUsed._isBigNumber)
    res.miner.should.be.a('string')
    res.extraData.should.be.a('string')
    res.transactions.should.be.a('array')
    assert(res.transactions.length === 0)
  })

  // TODO(mark): subject to change
  it('should getBlockWithTransactions', async () => {
    const res = await provider.getBlockWithTransactions(0)
    res.should.be.a('object')
  })
})
