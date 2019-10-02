/* External Imports */
import chai = require('chai')
import bignum = require('chai-bignumber')
import { solidity } from 'ethereum-waffle'

chai.use(bignum())
chai.use(solidity)
const should = chai.should()
const expect = chai.expect

export { should, expect }
