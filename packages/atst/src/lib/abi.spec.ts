import { describe, expect, it } from 'vitest'

import { abi } from './abi'

/**
 * This is a low value test that I made only because
 * it makes for a good final check that indeed are
 * exporting the correct abi
 */
describe('abi', () => {
  it('is the correct abi', () => {
    const methodNames = abi.map((obj) => (obj as { name: string }).name)
    expect(methodNames).toMatchInlineSnapshot(`
          [
            undefined,
            "AttestationCreated",
            "attest",
            "attest",
            "attestations",
            "version",
          ]
        `)
  })
})
