/* External Imports */


// Create a defer function which will allow us to add our promise to the messageQueue
function defer() {
  const deferred = {
    promise: null,
    resolve: null,
    reject: null,
  }
  deferred.promise = new Promise((resolve, reject) => {
    deferred.resolve = resolve
    deferred.reject = reject
  })
  return deferred
}


/* Logging */
import debug from 'debug'
const log = debug('test:info:ownership-predicate')

import chai = require('chai')
import {createMockProvider, deployContract, getWallets, solidity} from 'ethereum-waffle';
import * as OwnershipPredicate from '../build/OwnershipPredicate.json'

chai.use(solidity);
const {expect} = chai;

describe.only('OwnershipPredicate', () => {
  const provider = createMockProvider()
  const [wallet, walletTo] = getWallets(provider)
  let ownershipPredicate

  beforeEach(async () => {
    ownershipPredicate = await deployContract(wallet, OwnershipPredicate, [])
  });

  it('checks if abi encoding works as expected', (done) => {
    const deferred = defer()
    let events = 0
    const numEvents = 6
    const logEvent = (event, otherthing) => {
      log('Result of event:')
      log(event)
      log('\n')
      events++
      if (events === numEvents) {
        done()
      }
    }
    ownershipPredicate.on('TestEncoding', logEvent)
    ownershipPredicate.on('TestEncoding2', logEvent)
    ownershipPredicate.on('TestEncoding3', logEvent)
    const res = ownershipPredicate.testEncoding()
    deferred
  });
});
