/* External Imports */
import chai = require('chai')
import chaiAsPromised from 'chai-as-promised'
import { solidity } from 'ethereum-waffle'

chai.use(solidity)
chai.use(chaiAsPromised)
const expect = chai.expect

export { expect }
