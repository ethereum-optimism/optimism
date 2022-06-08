import chai = require('chai')
import chaiAsPromised from 'chai-as-promised'

// Chai plugins go here.
chai.use(chaiAsPromised)

const should = chai.should()
const expect = chai.expect

export { should, expect }
