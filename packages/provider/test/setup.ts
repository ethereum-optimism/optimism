import chai = require('chai')
import chaiAsPromised = require('chai-as-promised')
import assert = require('assert')

chai.use(chaiAsPromised)
chai.should()

export { assert }
