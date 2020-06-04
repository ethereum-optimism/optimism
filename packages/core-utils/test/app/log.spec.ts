import { getLogger, logError } from '../../src/app'
import { Logger } from '../../src/types'

let lastDebug
const FakeDebug = (identifier) => {
  return (...logs: any[]) => {
    lastDebug = `${identifier} ${logs.join(' ')}`
  }
}

describe('Logger Tests', () => {
  let log: Logger
  beforeEach(() => {
    log = getLogger('derp', true, FakeDebug)
    lastDebug = undefined
  })

  it('logs single line non-error', () => {
    const logString = '\n test \n'
    logString
      .indexOf('\n')
      .should.eq(0, 'Cannot find newline when it should be at pos 0')

    lastDebug = undefined
    log.debug(logString)
    lastDebug.indexOf('\n').should.equal(-1, 'Log line has multiple lines!')
  })

  it('logs single line error', () => {
    const e = new Error()
    e.stack.indexOf('\n').should.be.gt(-1, 'No new line in stack trace!')

    lastDebug = undefined
    logError(log, `some error`, e)
    lastDebug.indexOf('\n').should.eq(-1, 'Stack trace has new line!')
  })

  it('logs objects in single line', () => {
    const obj = {
      test: 'yes',
      question: 'answer',
      bools: true,
      numbers: 735,
      somethingVeryLong:
        'ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok',
    }
    lastDebug = undefined
    log.debug(obj)
    lastDebug.indexOf('\n').should.equal(-1, 'Log line has multiple lines!')
  })

  it('logs many different things in single line', () => {
    const obj = {
      test: 'yes',
      question: 'answer',
      bools: true,
      numbers: 735,
      somethingVeryLong:
        'ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok ok',
    }
    const e = new Error('some error here')
    const string = '\n test \n'
    lastDebug = undefined
    log.debug(obj, e, string)
    lastDebug.indexOf('\n').should.equal(-1, 'Log line has multiple lines!')
  })
})
