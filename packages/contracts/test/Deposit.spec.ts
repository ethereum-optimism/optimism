/* Logging */
import debug from 'debug'
const log = debug('test:info:state-ownership')

import chai = require('chai')
import {createMockProvider, deployContract, getWallets, solidity} from 'ethereum-waffle';
import * as BasicTokenMock from '../build/BasicTokenMock.json'
import * as Deposit from '../build/Deposit.json'

chai.use(solidity);
const {expect} = chai;

describe.only('INTEGRATION: Example', () => {
  const provider = createMockProvider()
  const [wallet, walletTo] = getWallets(provider)
  let token
  let depositContract

  beforeEach(async () => {
    token = await deployContract(wallet, BasicTokenMock, [wallet.address, 1000])
    depositContract = await deployContract(wallet, Deposit, [token.address])
  });

  it('allows deposit to be called after approving erc20 movement', async () => {
    await token.approve(depositContract.address, 500)
    await depositContract.deposit(123, { predicateAddress: '0xF6c105ED2f0f5Ffe66501a4EEdaD86E10df19054', data: '0x1234' })
    log('success')
  });
});
