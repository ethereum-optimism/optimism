import '../../common/setup'

/* External Imports */
import { deployContract } from 'ethereum-waffle'
import { ethers, Wallet, Contract } from 'ethers'
import { ecsign } from 'ethereumjs-util'

/* Internal Imports */
import { waffle } from '../../../src/waffle'

/* Contract Imports */
import * as Precompiles from '../../temp/build/waffle/Precompiles.json'

describe('Precompile Support', () => {
  let wallet: Wallet
  let provider: any
  beforeEach(async () => {
    provider = new waffle.MockProvider()
    ;[wallet] = provider.getWallets()
  })

  let precompiles: Contract
  beforeEach(async () => {
    precompiles = await deployContract(wallet, Precompiles, [])
  })

  it('should correctly ecrecover signer address', async () => {
    const message = '0xdeadbeef'
    const hashedMessage = ethers.utils.keccak256(message)
    const { v, r, s } = ecsign(
      Buffer.from(hashedMessage.slice(2), 'hex'),
      Buffer.from(wallet.privateKey.slice(2), 'hex')
    )
    const recoveredAddress = await precompiles.recoverAddr(
      hashedMessage,
      v,
      r,
      s
    )
    recoveredAddress.should.equal(wallet.address)
  })

  it('should correctly calculate SHA256 hash', async () => {
    const message = '0xdeadbeef'
    const expectedHash = ethers.utils.sha256(message)
    const hash = await precompiles.calculateSHA256(message)
    hash.should.equal(expectedHash)
  })

  it('should correctly calldataCopy', async () => {
    const message = '0xdeadbeef'
    await precompiles.calldataCopy(message)
    const copiedMessage = await precompiles.copiedData()
    copiedMessage.should.equal(message)
  })

  it('bigmodexp', async () => {
    const base = 2
    const exp = 257
    const mod = 13
    const result = await precompiles.expmod(base, exp, mod)
    const expectedResult = Math.pow(base, exp) % mod
    result.toNumber().should.equal(expectedResult)
  })
})
