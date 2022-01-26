import { expect } from '@eth-optimism/core-utils/test/setup'
import { ethers, BigNumber } from 'ethers'
import { Genesis } from '@eth-optimism/core-utils/src/types'
import {
  remove0x,
  add0x,
} from '@eth-optimism/core-utils/src/common/hex-strings'
import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'

import { GenesisJsonProvider } from './provider'

const account = '0x66a84544bed4ca45b3c024776812abf87728fbaf'

const genesis: Genesis = {
  config: {
    chainId: 0,
    homesteadBlock: 0,
    eip150Block: 0,
    eip155Block: 0,
    eip158Block: 0,
    byzantiumBlock: 0,
    constantinopleBlock: 0,
    petersburgBlock: 0,
    istanbulBlock: 0,
    muirGlacierBlock: 0,
    clique: {
      period: 0,
      epoch: 0,
    },
  },
  difficulty: '0x',
  gasLimit: '0x',
  extraData: '0x',
  alloc: {
    [remove0x(account)]: {
      nonce: 101,
      balance: '234',
      codeHash: ethers.utils.keccak256('0x6080'),
      root: '0x',
      code: '6080',
      storage: {
        '0000000000000000000000000000000000000000000000000000000000000002':
          '989680',
        '0000000000000000000000000000000000000000000000000000000000000003':
          '536f6d65205265616c6c7920436f6f6c20546f6b656e204e616d650000000036',
        '7d55c28652d09dd36b33c69e81e67cbe8d95f51dc46ab5b17568d616d481854d':
          '989680',
      },
    },
  },
}

describe('GenesisJsonProvider', () => {
  let provider
  before(() => {
    provider = new GenesisJsonProvider(genesis)
  })

  it('should get nonce', async () => {
    const nonce = await provider.getTransactionCount(account)
    expect(nonce).to.deep.eq(101)
  })

  it('should get nonce on missing account', async () => {
    const nonce = await provider.getTransactionCount('0x')
    expect(nonce).to.deep.eq(0)
  })

  it('should get code', async () => {
    const code = await provider.getCode(account)
    expect(code).to.deep.eq('0x6080')
  })

  it('should get code on missing account', async () => {
    const code = await provider.getCode('0x')
    expect(code).to.deep.eq('0x')
  })

  it('should get balance', async () => {
    const balance = await provider.getBalance(account)
    expect(balance.toString()).to.deep.eq(BigNumber.from(234).toString())
  })

  it('should get balance on missing account', async () => {
    const balance = await provider.getBalance('0x')
    expect(balance.toString()).to.deep.eq('0')
  })

  it('should get storage', async () => {
    const storage = await provider.getStorageAt(account, 2)
    expect(storage).to.deep.eq('0x989680')
  })

  it('should get storage of missing account', async () => {
    const storage = await provider.getStorageAt('0x', 0)
    expect(storage).to.deep.eq('0x')
  })

  it('should get storage of missing slot', async () => {
    const storage = await provider.getStorageAt(account, 9999999999999)
    expect(storage).to.deep.eq('0x')
  })

  it('should call eth_getProof', async () => {
    const proof = await provider.send('eth_getProof', [account])
    // There is code at the account, so it shouldn't be the null code hash
    expect(proof.codeHash).to.not.eq(add0x(KECCAK256_NULL_S))
    // There is storage so it should not be the null storage hash
    expect(proof.storageHash).to.not.eq(add0x(KECCAK256_RLP_S))
  })

  it('should call eth_getProof on missing account', async () => {
    const proof = await provider.send('eth_getProof', ['0x'])
    expect(proof.codeHash).to.eq(add0x(KECCAK256_NULL_S))
    expect(proof.storageHash).to.eq(add0x(KECCAK256_RLP_S))
  })

  it('should also initialize correctly with state dump', async () => {
    provider = new GenesisJsonProvider(genesis.alloc)
    expect(provider).to.be.instanceOf(GenesisJsonProvider)
  })
})
