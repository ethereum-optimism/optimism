import { validators } from '../dist'
import { BaseServiceV2 } from '../src'

type ServiceOptions = {
  camelCase: string
}

class Service extends BaseServiceV2<ServiceOptions, {}, {}> {
  constructor(options?: Partial<ServiceOptions>) {
    super({
      name: 'test-service',
      version: '0.0',
      options,
      optionsSpec: {
        camelCase: { validator: validators.str, desc: 'test' },
      },
      metricsSpec: {},
    })
  }
  protected async main() {
    /* eslint-disable @typescript-eslint/no-empty-function */
  }
}

describe('BaseServiceV2', () => {
  it('base service ctor does not throw on camel case options', async () => {
    new Service({ camelCase: 'test' })
  })
})
