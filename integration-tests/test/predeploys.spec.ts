import { expect } from 'chai'
import { ethers } from 'hardhat'

/* Imports: External */
import { Contract, Wallet } from 'ethers'
import { OptimismEnv } from './shared/env'
import { getContractInterface } from '@eth-optimism/contracts'

describe('ECDSAContractAccount', () => {
  let l2Wallet: Wallet

  const DEFAULT_TRANSACTION = {
    to: '0x' + '1234'.repeat(10),
    gasLimit: 33600000000001,
    gasPrice: 0,
    data: '0x',
    value: 0,
  }
  before(async () => {
    const env = await OptimismEnv.new()
    l2Wallet = env.l2Wallet
  })

  let ProxyEOA: Contract

  beforeEach(async () => {
    // Send a transaction to ensure there is a ProxyEOA deployed at l2Wallet.address
    const result = await l2Wallet.sendTransaction(DEFAULT_TRANSACTION)
    await result.wait()
  })

  it('should correctly evaluate isValidSignature', async () => {
    ProxyEOA = new Contract(
      l2Wallet.address,
      getContractInterface('OVM_ECDSAContractAccount'),
      l2Wallet
    )
    const message = '0x42'
    const messageHash = ethers.utils.hashMessage(message)
    const signature = await l2Wallet.signMessage(message)
    const isValid = await ProxyEOA.isValidSignature(messageHash, signature)
    expect(isValid).to.equal('0x1626ba7e')
  })
})
