import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';

import { Direction, Relayer } from './shared/watcher-utils'

import L1MessageJson from '../artifacts/contracts/Message/L1Message.sol/L1Message.json'
import L2MessageJson from '../artifacts-ovm/contracts/Message/L2Message.sol/L2Message.json'

import { OptimismEnv } from './shared/env'

import * as fs from 'fs'

describe('Messenge Relayer Test', async () => {

  let L1Message: Contract
  let L2Message: Contract

  let env: OptimismEnv

  before(async () => {

    const addressData = fs.readFileSync('./deployment/local/addresses.json', 'utf8')
    const addressArray = JSON.parse(addressData)

    env = await OptimismEnv.new()

    L1Message = new Contract(
      addressArray.L1Message,
      L1MessageJson.abi,
      env.bobl1Wallet
    )

    L2Message = new Contract(
      addressArray.L2Message,
      L2MessageJson.abi,
      env.bobl2Wallet
    )

    const accountNonceBob1 = await env.l1Provider.getTransactionCount(env.bobl1Wallet.address)
    console.log(`accountNonceBob1:`,accountNonceBob1)

    const accountNonceBob2 = await env.l2Provider.getTransactionCount(env.bobl2Wallet.address)
    console.log(`accountNonceBob2:`,accountNonceBob2)

  })
  
  it('should send message from L2 to L1', async () => {
    await env.waitForXDomainTransaction(
      L2Message.sendMessageL2ToL1({
        gasLimit: 800000, 
        gasPrice: 0
      }),
      Direction.L2ToL1
    )
  })

  it('should send message from L1 to L2', async () => {
    await env.waitForXDomainTransaction(
      L1Message.sendMessageL1ToL2(),
      Direction.L1ToL2
    )
  })
})