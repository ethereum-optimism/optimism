/**
 * Copyright 2020, Optimism PBC
 * MIT License
 * https://github.com/ethereum-optimism
 */

import './setup'

/* Imports: External */
import { Web3Provider } from '@ethersproject/providers'
import { ganache } from '@eth-optimism/ovm-toolchain'
import { verifyMessage } from '@ethersproject/wallet'

/* Imports: Internal */
import { OptimismProvider } from '../src/index'

describe('Signer', () => {
  let provider: OptimismProvider
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
