/* Imports: Internal */
import { expect } from '../setup'
import { sleep, clone, reqenv, getenv } from '../../src'

describe('sleep', async () => {
  it('should return wait input amount of ms', async () => {
    const startTime = Date.now()
    await sleep(1000)
    const endTime = Date.now()
    expect(startTime + 1000 <= endTime).to.deep.equal(true)
  })
})

describe('clone', async () => {
  it('should return a cloned object', async () => {
    const exampleObject = { example: 'Example' }
    const clonedObject = clone(exampleObject)
    expect(clonedObject).to.not.equal(exampleObject)
    expect(JSON.stringify(clonedObject)).to.equal(JSON.stringify(exampleObject))
  })
})

describe('reqenv', async () => {
  let cachedEnvironment: NodeJS.ProcessEnv
  const temporaryEnvironmentKey = 'testVariable'
  const temporaryEnvironment = {
    [temporaryEnvironmentKey]: 'This is an environment variable',
  }

  before(() => {
    cachedEnvironment = process.env
    process.env = temporaryEnvironment
  })

  it('should return an existent environment variable', async () => {
    const requiredEnvironmentValue = reqenv(temporaryEnvironmentKey)
    expect(requiredEnvironmentValue).to.equal(
      temporaryEnvironment[temporaryEnvironmentKey]
    )
  })

  it('should throw an error trying to return a variable that does not exist', async () => {
    const undeclaredVariableName = 'undeclaredVariable'
    const failedReqenv = () => reqenv(undeclaredVariableName)
    expect(failedReqenv).to.throw()
  })

  after(() => {
    process.env = cachedEnvironment
  })
})

describe('getenv', async () => {
  let cachedEnvironment: NodeJS.ProcessEnv
  const temporaryEnvironmentKey = 'testVariable'
  const temporaryEnvironment = {
    [temporaryEnvironmentKey]: 'This is an environment variable',
  }
  const fallback = 'fallback'

  before(() => {
    cachedEnvironment = process.env
    process.env = temporaryEnvironment
  })

  it('should return an existent environment variable', async () => {
    const environmentVariable = getenv(temporaryEnvironmentKey)
    expect(environmentVariable).to.equal(
      temporaryEnvironment[temporaryEnvironmentKey]
    )
  })

  it('should return an existent environment variable even if fallback is passed', async () => {
    const environmentVariable = getenv(temporaryEnvironmentKey, fallback)
    expect(environmentVariable).to.equal(
      temporaryEnvironment[temporaryEnvironmentKey]
    )
  })

  it('should return fallback if variable is not defined', async () => {
    const undeclaredVariableName = 'undeclaredVariable'
    expect(getenv(undeclaredVariableName, fallback)).to.equal(fallback)
  })

  it('should return undefined if no fallback is passed and variable is not defined', async () => {
    expect(getenv('undeclaredVariable')).to.be.undefined
  })

  after(() => {
    process.env = cachedEnvironment
  })
})
