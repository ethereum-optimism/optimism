import { expect } from '../setup'
import { BaseServiceV2, StandardOptions, validators } from '../../src'

interface TestServiceOptions {
  foo: string
  bar: number
}

class TestService extends BaseServiceV2<TestServiceOptions, any, any> {
  constructor(options: Partial<TestServiceOptions & StandardOptions>) {
    super({
      name: 'test-service',
      version: '1.0.0',
      options,
      optionsSpec: {
        foo: {
          validator: validators.str,
          desc: 'a string value',
        },
        bar: {
          validator: validators.num,
          desc: 'a number value',
        },
      },
      metricsSpec: {},
    })
  }

  async main() {
    // No-op
  }
}

describe('BaseServiceV2', () => {
  describe('constructor', () => {
    describe('validation', () => {
      it('should parse when all configuration is valid', () => {
        const service = new TestService({
          foo: 'abc',
          bar: 123,
        })

        expect(service.options.foo).to.equal('abc')
        expect(service.options.bar).to.equal(123)
      })

      it('should throw an error when a value is invalid', () => {
        expect(() => {
          new TestService({
            foo: 'abc',
            bar: '123' as any,
          })
        }).to.throw()
      })

      it('should throw an error when more than one value is invalid', () => {
        expect(() => {
          new TestService({
            foo: 123 as any,
            bar: '123' as any,
          })
        }).to.throw()
      })

      it('should throw an error when a value is missing', () => {
        expect(() => {
          new TestService({
            bar: 123,
          })
        }).to.throw()
      })

      it('should throw an error when an extra value is given', () => {
        expect(() => {
          new TestService({
            foo: 'abc',
            bar: 123,
            baz: 'xyz',
          } as any)
        }).to.throw()
      })
    })
  })
})
