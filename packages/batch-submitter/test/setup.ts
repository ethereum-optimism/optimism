/* External Imports */
import chai = require('chai')
import sinonChai from 'sinon-chai'
import Mocha from 'mocha'

const should = chai.should()
const expect = chai.expect
chai.use(sinonChai)

export { should, expect, chai, Mocha }
