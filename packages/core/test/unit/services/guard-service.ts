import '../../setup'

/* Internal Imports */
import { GuardService } from '../../../src/services'
import { createApp } from '../../mock'

describe('GuardService', () => {
  const { app } = createApp()
  const guard = new GuardService({ app, name: 'guard' })

  beforeEach(async () => {
    await guard.start()
  })

  afterEach(async () => {
    await guard.stop()
  })

  it('should have dependencies', () => {
    const dependencies = ['eventHandler']
    guard.dependencies.should.deep.equal(dependencies)
  })

  it('should have a name', () => {
    guard.name.should.equal('guard')
  })

  it('should start correctly', () => {
    guard.started.should.be.true
  })
})
