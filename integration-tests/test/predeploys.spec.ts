import { expect } from 'chai'
import { ethers } from 'hardhat'

/* Imports: External */
import { Contract, Wallet } from 'ethers'
import { OptimismEnv } from './shared/env'
import { DEFAULT_TRANSACTION } from './shared/utils'
import { getContractInterface } from '@eth-optimism/contracts'

describe('ECDSAContractAccount', () => {
  let l2Wallet: Wallet

  before(async () => {
    const env = await OptimismEnv.new()
    l2Wallet = env.l2Wallet
  })

  let ProxyEOA: Contract
  let messageHash: string
  let signature: string

  before(async () => {
    // Send a transaction to ensure there is a ProxyEOA deployed at l2Wallet.address
    const result = await l2Wallet.sendTransaction(DEFAULT_TRANSACTION)
    await result.wait()
    ProxyEOA = new Contract(
      l2Wallet.address,
      getContractInterface('OVM_ECDSAContractAccount'),
      l2Wallet
    )
    const message = '0x42'
    messageHash = ethers.utils.hashMessage(message)
    signature = await l2Wallet.signMessage(message)
  })

  it('should correctly evaluate isValidSignature from this wallet', async () => {
    const isValid = await ProxyEOA.isValidSignature(messageHash, signature)
    expect(isValid).to.equal('0x1626ba7e')
  })

  it('should correctly evaluate isValidSignature from other wallet', async () => {
    const otherWallet = Wallet.createRandom().connect(l2Wallet.provider)
    const isValid = await ProxyEOA.connect(otherWallet).isValidSignature(
      messageHash,
      signature
    )
    expect(isValid).to.equal('0x1626ba7e')
  })
})
