/* External Imports */
import fs = require('fs')
import path = require('path')
import chai = require('chai')
import chaiAsPromised = require('chai-as-promised')

/* Internal Imports */
import { rootPath } from '../index'

chai.use(chaiAsPromised)
const should = chai.should()
