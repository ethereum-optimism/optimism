/**
 * Optimism Copyright 2020.
 */

import { JsonRpcServer } from '@eth-optimism/core-utils'
import { Web3Provider } from '@ethersproject/providers'
import chai = require('chai')
import chaiAsPromised = require('chai-as-promised')
import { ganache } from '@eth-optimism/ovm-toolchain'
import BigNumber = require('bn.js')
import { OptimismProvider, serializeEthSignTransaction } from '../src/index'
import { verifyMessage } from '@ethersproject/wallet'

chai.use(chaiAsPromised)
const should = chai.should()

describe('sendTransaction', () => {
  let provider
  let server

  const handlers = {
    eth_chainId: () => '0x1'
  }

  before(async () => {
    const web3 = new Web3Provider(ganache.provider({}))
    provider = new OptimismProvider('http://localhost:3000', web3)

    server = new JsonRpcServer(handlers, 'localhost', 3000)
    await server.listen()
  })

  after(async () => {
    await server.close()
  })

  it('should sign transaction', async () => {
    const tx = {
      to: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
      nonce: 0,
      gasLimit: new BigNumber(0),
      gasPrice: new BigNumber(0),
      data: '0x00',
      value: new BigNumber(0),
      chainId: 1
    }

    const signer = provider.getSigner();
    const address = await signer.getAddress()
    const sig = await signer.signTransaction(tx)

    const message = serializeEthSignTransaction(tx)
    const recovered = verifyMessage(message, sig)

    address.should.eq(recovered)
  })
})
