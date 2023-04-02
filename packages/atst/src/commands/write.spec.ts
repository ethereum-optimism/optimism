import { Address } from '@wagmi/core'
import { Wallet } from 'ethers'
import { describe, expect, it } from 'vitest'

import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { read } from './read'
import { write } from './write'

describe(`cli:${write.name}`, () => {
  it('should write attestation', async () => {
    // Anvil account[0]
    const privateKey =
      '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80'
    const publicKey = '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266'

    const about = Wallet.createRandom().address as Address
    const key = 'key'
    const value = 'value'
    const rpcUrl = 'http://localhost:8545'

    const txHash = await write({
      privateKey,
      about,
      key,
      value,
      contract: ATTESTATION_STATION_ADDRESS,
      rpcUrl,
      dataType: 'string',
    })

    expect(txHash.startsWith('0x')).toBe(true)

    // check that attestation was written
    const attestation = await read({
      creator: publicKey,
      about,
      key,
      dataType: 'string',
      contract: ATTESTATION_STATION_ADDRESS,
      rpcUrl,
    })

    expect(attestation).toBe(value)
  })
})
