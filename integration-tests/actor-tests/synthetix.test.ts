import { BigNumber, Contract, utils, Wallet, ContractFactory } from 'ethers'
import { actor, run, setupActor, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'
import * as path from 'path'

import {
  compileInstance,
  deployInstance,
  connectInstances,
} from './synthetix/test/integration/utils/deploy'

interface Context {
  contracts: { [name: string]: Contract }
  wallet: Wallet
}

const BUILD_FOLDER = 'build'

actor('Synthetix Trader', () => {
  let env: OptimismEnv

  setupActor(async () => {
    env = await OptimismEnv.new()

    // TODO: how to know what the addresses are...

    // TODO: these constants are already defined someplace
    const providerUrl = 'http://localhost'
    const providerPortL1 = '9545'
    const providerPortL2 = '8545'

    // TODO: don't build everytime
    const buildPathEvm = path.join(__dirname, BUILD_FOLDER)
    const buildPathOvm = path.join(__dirname, `${BUILD_FOLDER}-ovm`)
    await compileInstance({ useOvm: false, buildPath: buildPathEvm })
    await compileInstance({ useOvm: true, buildPath: buildPathOvm })

    await deployInstance({
      useOvm: false,
      providerUrl,
      providerPort: providerPortL1,
      buildPath: buildPathEvm,
    })

    await deployInstance({
      useOvm: true,
      providerUrl,
      providerPort: providerPortL2,
      buildPath: buildPathOvm,
    })

    await connectInstances({
      providerUrl,
      providerPortL1,
      providerPortL2,
    })


  })

  setupRun(async () => {
    const wallet = Wallet.createRandom().connect(env.l2Provider)

    await env.l2Wallet.sendTransaction({
      to: wallet.address,
      value: utils.parseEther('0.1'),
    })

    return {
      contracts: {},
      wallet,
    }
  })

  run(async (b, ctx: Context) => {
    await b.bench('swap', async () => {
      let a = ctx
      a = a
    })
  })
})
