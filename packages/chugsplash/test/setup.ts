/* External Imports */
import chai = require('chai')
import Mocha from 'mocha'
import { solidity } from 'ethereum-waffle'
import chaiAsPromised = require('chai-as-promised')

chai.use(solidity)
chai.use(chaiAsPromised)
const should = chai.should()
const expect = chai.expect

export { should, expect, Mocha }
