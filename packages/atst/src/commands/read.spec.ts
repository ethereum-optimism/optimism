import { describe, expect, it } from 'vitest'

import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { watchConsole } from '../test/watchConsole'
import { read } from './read'

describe(`cli:${read.name}`, () => {
  it('should read attestation', async () => {
    const creator = '0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3'
    const about = '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5'
    const key = 'optimist.base-uri'
    const dataType = 'string'

    const consoleUtil = watchConsole()

    await read({
      creator,
      about,
      key,
      dataType,
      contract: ATTESTATION_STATION_ADDRESS,
      rpcUrl: 'http://localhost:8545',
    })
    expect(consoleUtil.formatted).toMatchInlineSnapshot(
      '"[37mhttps://assets.optimism.io/4a609661-6774-441f-9fdb-453fdbb89931-bucket/optimist-nft/attributes[39m"'
    )
  })
})
