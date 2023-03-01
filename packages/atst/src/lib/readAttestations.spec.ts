import { createClient } from '@wagmi/core'
import { providers } from 'ethers'
import { expect, describe, it } from 'vitest'

import { readAttestation } from './readAttestation'
import { readAttestations } from './readAttestations'

const creator = '0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3'
const about = '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5'
const key = 'optimist.base-uri'

const provider = new providers.JsonRpcProvider({
  url: 'http://localhost:8545',
})

createClient({
  provider,
})

describe(readAttestation.name, () => {
  it('should return attestations from attestation station', async () => {
    const dataType = 'string'

    const result = await readAttestations(
      {
        creator,
        about,
        key,
        dataType,
      },
      {
        creator,
        about,
        key,
        dataType: 'bool',
      },
      {
        creator,
        about,
        key,
        dataType: 'bytes',
      },
      {
        creator,
        about,
        key,
        dataType: 'number',
      }
    )

    expect(result).toMatchInlineSnapshot(
      `
      [
        "https://assets.optimism.io/4a609661-6774-441f-9fdb-453fdbb89931-bucket/optimist-nft/attributes",
        true,
        "0x68747470733a2f2f6173736574732e6f7074696d69736d2e696f2f34613630393636312d363737342d343431662d396664622d3435336664626238393933312d6275636b65742f6f7074696d6973742d6e66742f61747472696275746573",
        {
          "hex": "0x68747470733a2f2f6173736574732e6f7074696d69736d2e696f2f34613630393636312d363737342d343431662d396664622d3435336664626238393933312d6275636b65742f6f7074696d6973742d6e66742f61747472696275746573",
          "type": "BigNumber",
        },
      ]
    `
    )
  })
})
