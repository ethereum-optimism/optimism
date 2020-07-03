import './setup'

/* External Imports */
import { add0x } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { addHandlerToProvider } from '@eth-optimism/rollup-full-node'
import { Contract, Wallet } from 'ethers'
import { getAddress, keccak256, solidityPack, sha256 } from 'ethers/utils'
import { ecsign } from 'ethereumjs-util'

/* Contract Imports */
import * as Precompiles from '../build/Precompiles.json'

describe('Precompiles', () => {
  let wallet
  let precompiles: Contract
  let provider

  beforeEach(async () => {
    provider = await createMockProvider()
    if (process.env.MODE === 'OVM') {
      provider = await addHandlerToProvider(provider)
    }
    const wallets = await getWallets(provider)
    wallet = wallets[1]
    precompiles = await deployContract(wallet, Precompiles, [])
  })

  it('should correctly ecrecover signer address', async () => {
    const message = '0xdeadbeef'
    const hashedMessage = keccak256(message)
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
    const expectedHash = sha256(message)
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
