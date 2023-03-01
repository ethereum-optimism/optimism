import { createClient } from '@wagmi/core'
import { providers } from 'ethers'
import { expect, describe, it } from 'vitest'

import { readAttestation } from './readAttestation'

const creator = '0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3'
const about = '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5'
const key = 'optimist.base-uri'
const dataType = 'string'

const provider = new providers.JsonRpcProvider({
  url: 'http://localhost:8545',
})

createClient({
  provider,
})

describe(readAttestation.name, () => {
  it('should return the attestation from attestation station', async () => {
    const result = await readAttestation(creator, about, key, dataType)

    expect(result).toMatchInlineSnapshot(
      '"https://assets.optimism.io/4a609661-6774-441f-9fdb-453fdbb89931-bucket/optimist-nft/attributes"'
    )
  })

  it('should throw an error if key is longer than 32 bytes', async () => {
    await expect(
      readAttestation(
        creator,
        about,
        'this is a key that is way longer than 32 bytes so this key should throw an error matching the inline snapshot',
        dataType
      )
    ).rejects.toThrowErrorMatchingInlineSnapshot(
      '"Key is longer than the max length of 32 for attestation keys"'
    )
  })
})

