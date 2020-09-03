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
import { parse } from '@ethersproject/transactions'
import { SignatureLike, joinSignature } from '@ethersproject/bytes'

import { mnemonic, etherbase } from './common'

chai.use(chaiAsPromised)
const should = chai.should()


describe('sendTransaction', () => {
  let provider
  let server

  const handlers = {
    eth_chainId: () => '0x1'
  }

  before(async () => {
    const web3 = new Web3Provider(ganache.provider({
      mnemonic
    }))
    provider = new OptimismProvider('http://127.0.0.1:8545', web3)
    //provider = new OptimismProvider('http://127.0.0.1:3000', web3)
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
    // Get the address represting the keypair used to sign the tx
    const address = await signer.getAddress()
    // Sign tx, get a RLP encoded hex string of the signed tx
    const signed = await signer.signTransaction(tx)
    // Decode the signed transaction
    const parsed = parse(signed)
    // Join the r, s and v values
    const sig = joinSignature(parsed as SignatureLike)
    // Serialize the transaction using the EthSign serialization
    const message = serializeEthSignTransaction(tx)
    // ecrecover and assert the addresses match
    const recovered = verifyMessage(message, sig)
    address.should.eq(recovered)
  })

  /*
  This depends on a running geth2 node or the endpoint
  being added to optimism-ganache

  it('should sendRawEthSignTransaction', async () => {
    const signer = provider.getSigner();
    const chainId = await signer.getChainId();

    const tx = {
      to: etherbase,
      nonce: 0,
      gasLimit: 21004,
      gasPrice: 100,
      data: '0x00',
      value: 10,
      chainId
    }

    // this isn't preserving the gasPrice and gasLimit..
    const hex = await signer.signTransaction(tx)

    // This incorrectly calculates "from" since it
    // uses EIP155 signature hashing.
    const address = await signer.getAddress()
    //const signed = parse(hex)
    //signed.from = address

    const result = await provider.send('eth_sendRawEthSignTransaction', [hex])
    console.log(result)
  })
  */
})
