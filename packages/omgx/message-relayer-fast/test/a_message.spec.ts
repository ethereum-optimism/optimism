import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';

import { Direction } from './shared/watcher-utils'

import L1MessageJson from '../contracts/L1Message.json'
import L2MessageJson from '../contracts/L2Message.json'

import { OptimismEnv } from './shared/env'
import * as fs from 'fs'

describe('Fast Messenge Relayer Test', async () => {

  let L1Message: Contract
  let L2Message: Contract

  let env: OptimismEnv

  before(async () => {

    env = await OptimismEnv.new()

    L1Message = new Contract(
      env.addressesOMGX.L1Message,
      L1MessageJson.abi,
      env.bobl1Wallet
    )

    L2Message = new Contract(
      env.addressesOMGX.L2Message,
      L2MessageJson.abi,
      env.bobl2Wallet
    )
  })

  it('should send message from L1 to L2', async () => {
    await env.waitForXDomainTransaction(
      L1Message.sendMessageL1ToL2(),
      Direction.L1ToL2
    )
  })

  it('should QUICKLY send message from L2 to L1 using the fast relayer', async () => {
    await env.waitForXDomainTransactionFast(
      L2Message.sendMessageL2ToL1({ gasLimit: 800000, gasPrice: 0 }),
      Direction.L2ToL1
    )
  })

})