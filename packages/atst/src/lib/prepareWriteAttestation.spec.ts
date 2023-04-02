import { connect, createClient } from '@wagmi/core'
import { providers, Wallet } from 'ethers'
import { expect, describe, it, beforeAll } from 'vitest'
import { MockConnector } from '@wagmi/core/connectors/mock'

import { prepareWriteAttestation } from './prepareWriteAttestation'
import { readAttestation } from './readAttestation'

const creator = '0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3'
const about = '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5'
const key = 'optimist.base-uri'

const chainId = 10

const provider = new providers.JsonRpcProvider(
  {
    url: 'http://localhost:8545',
  },
  chainId
)

const wallet = Wallet.createRandom({ provider })

createClient({
  provider,
})

beforeAll(async () => {
  await connect({
    connector: new MockConnector({
      options: {
        chainId,
        signer: new Wallet(wallet.privateKey, provider),
      },
    }),
  })
})

describe(prepareWriteAttestation.name, () => {
  it('Should correctly prepare an attestation', async () => {
    const result = await prepareWriteAttestation(about, key, 'hello world')

    expect(result.address).toMatchInlineSnapshot(
      '"0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77"'
    )
    expect(result.chainId).toMatchInlineSnapshot('undefined')
    expect(result.functionName).toMatchInlineSnapshot('"attest"')
    expect(result.mode).toMatchInlineSnapshot('"prepared"')
    expect(result.request.gasLimit).toMatchInlineSnapshot(`
      {
        "hex": "0xd6c9",
        "type": "BigNumber",
      }
    `)
  })

  it('should work for key longer than 32 bytes', async () => {
    const dataType = 'string'

    expect(
      await readAttestation(
        creator,
        about,
        'this is a key that is way longer than 32 bytes so this key should throw an error matching the inline snapshot',
        dataType
      )
    ).toMatchInlineSnapshot('""')
  })
})
