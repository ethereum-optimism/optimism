/**
 * Copyright 2020, Optimism PBC
 * MIT License
 * https://github.com/ethereum-optimism
 */

import { isHexString } from '@eth-optimism/core-utils'
import { Web3Provider } from '@ethersproject/providers'
import { OptimismProvider } from '../src/index'
import chai = require('chai')
import chaiAsPromised = require('chai-as-promised')
import { ganache } from '@eth-optimism/ovm-toolchain'
import { verifyMessage } from '@ethersproject/wallet'

chai.use(chaiAsPromised)
const should = chai.should()

describe('Signer', () => {
  let provider

  before(() => {
    const web3 = new Web3Provider(ganache.provider({}))
    provider = new OptimismProvider('http://localhost:3000', web3)
  })

  it('should sign message', async () => {
    const signer = provider.getSigner()
    const addr = await signer.getAddress()

    const message = 'foobar'
    const sig = await signer.signMessage(message)
    const recovered = verifyMessage(message, sig)

    recovered.should.eq(addr)
  })
})
