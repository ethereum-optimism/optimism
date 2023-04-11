import { writeContract } from '@wagmi/core'
import { describe, expect, it } from 'vitest'

import { writeAttestation } from './writeAttestation'

describe(writeAttestation.name, () => {
  it('rexports writeContract from @wagmi/core', () => {
    expect(writeAttestation).toBe(writeContract)
  })
})
