import { Contract, utils, Wallet } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { actor, run, setupActor, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'
import { reqenv, getenv } from '@eth-optimism/core-utils'
import fetch from 'node-fetch'
import snx from 'synthetix'

// TODO: move dockerfile + docker entrypoint to the
// ops directory and then add docs to the readme
// on how to run it

const vars = {
  CONTRACTS_JSON_URL: getenv('CONTRACTS_JSON_URL'),
  SYNTHETIX_SOURCE_NETWORK: getenv('SYNTHETIX_SOURCE_NETWORK', 'mainnet'),
  SYNTHETIX_ADDRESS: getenv('SYNTHETIX_ADDRESS'),
  EXCHANGER_ADDRESS: getenv('EXCHANGER_ADDRESS'),
}

interface Context {
  contracts: { [name: string]: Contract }
  wallet: Wallet
}

const getSynthetixContract = (
  name: string,
  address: string,
  provider?: Provider
): Contract => {
  const source = snx.getSource({
    network: vars.SYNTHETIX_SOURCE_NETWORK,
    contract: name,
  })
  // TODO: debug
  if (!source) {
    console.log(name)
  }

  return new Contract(address, source.abi, provider)
}

actor('Synthetix Trader', () => {
  let env: OptimismEnv

  setupActor(async () => {
    env = await OptimismEnv.new()

    // fetch the contract addresses from the remote server
    if (vars.CONTRACTS_JSON_URL) {
      const response = await fetch(vars.CONTRACTS_JSON_URL)
      const data = await response.json()

      vars.SYNTHETIX_ADDRESS = data.l2.Synthetix
      vars.EXCHANGER_ADDRESS = data.l2.Exchanger
    } else {
      vars.CONTRACTS_JSON_URL = reqenv('CONTRACTS_JSON_URL')
      vars.SYNTHETIX_SOURCE_NETWORK = reqenv('SYNTHETIX_SOURCE_NETWORK')
    }

    // address, abi, signer
    // Create the contract objects here using the vars object
    const Synthetix = getSynthetixContract('Synthetix', vars.SYNTHETIX_ADDRESS)
    const Exchanger = getSynthetixContract('Exchanger', vars.EXCHANGER_ADDRESS)

    // TODO
    console.log(Synthetix)
    console.log(Exchanger)
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
      const balance = await ctx.wallet.getBalance()
      console.log(balance)
    })
  })
})
