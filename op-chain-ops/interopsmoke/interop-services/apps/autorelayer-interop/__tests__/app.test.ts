import type { Option } from 'commander'
import { describe, expect, it } from 'vitest'

import { ConfigSchema, RelayerApp } from '@/app.js'

const ALL_MODULES = [
  'eth-bridge',
  'general-relay',
  'promise',
  'callback-share',
]

describe('RelayerApp configuration defaults (R1)', () => {
  it('ConfigSchema defaults enabledModules to all four modules', () => {
    const parsed = ConfigSchema.parse({
      loopIntervalMs: 2000,
      ponderInteropApi: 'http://127.0.0.1:42069',
      sponsoredEndpoint: 'http://127.0.0.1:3000',
      senderPrivateKeys: [],
    })
    expect(parsed.enabledModules).toEqual(ALL_MODULES)
  })

  it('the --enabled-modules CLI option defaults to all four modules', () => {
    const app = new RelayerApp()
    const options = (
      app as unknown as { additionalOptions(): Option[] }
    ).additionalOptions()
    const enabledModulesOption = options.find(
      (o) => o.long === '--enabled-modules',
    )
    expect(enabledModulesOption).toBeDefined()
    expect(enabledModulesOption!.defaultValue).toEqual(ALL_MODULES)
  })
})
