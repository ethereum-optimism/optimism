import './setup'

import {runFullnode, deployContract, Web3RpcMethods, TestWeb3Handler } from '@eth-optimism/rollup-full-node'
import {Contract, Wallet} from 'ethers'
import {JsonRpcProvider, Provider} from 'ethers/providers'


const TimestampCheckerContract = require('../build/TimestampChecker.json')

const secondsSinceEopch = (): number => {
  return Math.round(Date.now() / 1000)
}

describe('Timestamp Checker', () => {
  let wallet: Wallet
  let timestampChecker: Contract
  let provider: JsonRpcProvider
  let fullnodeServer

  before(async () => {
    ;[fullnodeServer] = await runFullnode(true)
  })

  after(async () => {
    try {
      await fullnodeServer.close()
    } catch (e) {
      // don't do anything
    }
  })

  beforeEach(async () => {
    provider = new JsonRpcProvider('http://0.0.0.0:8545')
    wallet = new Wallet(Wallet.createRandom().privateKey, provider)
    const deployWallet = new Wallet(Wallet.createRandom().privateKey, provider)
    timestampChecker = await deployContract(deployWallet, TimestampCheckerContract, [], {})
  })

  it('should retrieve initial timestamp correctly', async () => {
    const timestamp = await timestampChecker.getTimestamp()

    timestamp.toNumber().should.equal(0, 'Timestamp mismatch!')
  })

  it('should retrieve the block timestamp correctly', async () => {
    const beforeTimestamp = secondsSinceEopch()
    const timestamp = (await timestampChecker.blockTimestamp()).toNumber()
    const afterTimestamp = secondsSinceEopch()

    const inequality = beforeTimestamp <= timestamp && timestamp <= afterTimestamp
    inequality.should.equal(true, 'Block timestamp mismatch!')
  })

  it('should retrieve the block timestamp correctly after increasing it', async () => {
    const previousTimestamp = (await timestampChecker.blockTimestamp()).toNumber()

    const increase: number = 9999
    const res = await provider.send(Web3RpcMethods.increaseTimestamp, [`0x${increase.toString(16)}`])
    res.should.equal(TestWeb3Handler.successString)

    const timestamp = (await timestampChecker.blockTimestamp()).toNumber()
    timestamp.should.be.gte(previousTimestamp + increase, '[Set] block timestamp mismatch!')
  })
  
})

