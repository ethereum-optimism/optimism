import './setup'

/* External Imports */
import { SimpleClient, getLogger, newInMemoryDB } from '@pigi/core'

/* Internal Imports */
import {
  AGGREGATOR_MNEMONIC,
  DummyRollupStateSolver,
  getGenesisState,
} from './helpers'
import {
  DefaultRollupStateMachine,
  UnipigTransitioner,
  RollupAggregator,
  RollupStateMachine,
  FaucetRequest,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  SignedStateReceipt,
  RollupClient,
  Balances,
} from '../src'

const log = getLogger('client-aggregator-integration', true)

/*********
 * TESTS *
 *********/

const timeout = 20_000
const testRecipientAddress = '0x7777b66b3C70137264BE7303812090EC42D85B4d'

describe('Mock Client/Aggregator Integration', () => {
  let accountAddress: string
  let aggregator: RollupAggregator
  let ovm: DummyRollupStateSolver
  let rollupClient: RollupClient
  let unipigWallet: UnipigTransitioner
  const walletPassword = 'Really great password'

  beforeEach(async function() {
    this.timeout(timeout)
    ovm = new DummyRollupStateSolver()
    rollupClient = new RollupClient(newInMemoryDB())
    unipigWallet = new UnipigTransitioner(newInMemoryDB(), ovm, rollupClient)

    // Now create a wallet account
    accountAddress = await unipigWallet.createAccount(walletPassword)

    const rollupStateMachine: RollupStateMachine = await DefaultRollupStateMachine.create(
      getGenesisState(accountAddress),
      newInMemoryDB()
    )

    // Initialize a mock aggregator
    await unipigWallet.unlockAccount(accountAddress, walletPassword)
    aggregator = new RollupAggregator(
      newInMemoryDB(),
      rollupStateMachine,
      'localhost',
      3000,
      AGGREGATOR_MNEMONIC
    )

    await aggregator.listen()
    // Connect to the mock aggregator
    rollupClient.connect(new SimpleClient('http://127.0.0.1:3000'))
  })

  afterEach(async () => {
    if (!!aggregator) {
      // Close the server
      await aggregator.close()
    }
  })

  describe('UnipigTransitioner', () => {
    it('should be able to query the aggregators balances', async () => {
      const response = await unipigWallet.getBalances(accountAddress)
      response.should.deep.equal({
        [UNI_TOKEN_TYPE]: 50,
        [PIGI_TOKEN_TYPE]: 50,
      })
    }).timeout(timeout)

    it('should return an error if the wallet tries to transfer money it doesnt have', async () => {
      try {
        await unipigWallet.send(
          UNI_TOKEN_TYPE,
          accountAddress,
          testRecipientAddress,
          10
        )
      } catch (err) {
        // Success!
      }
    }).timeout(timeout)

    it('should successfully transfer if alice sends money', async () => {
      await unipigWallet.send(
        UNI_TOKEN_TYPE,
        accountAddress,
        testRecipientAddress,
        10
      )
      const recipientBalances: Balances = await unipigWallet.getBalances(
        testRecipientAddress
      )
      recipientBalances[UNI_TOKEN_TYPE].should.equal(10)
    }).timeout(timeout)

    it('should successfully transfer if first faucet is requested', async () => {
      const newPassword = 'new address password'
      const newAddress = await unipigWallet.createAccount(newPassword)
      await unipigWallet.unlockAccount(newAddress, newPassword)

      // First collect some funds from the faucet
      await unipigWallet.requestFaucetFunds(newAddress, 10)
      const balances: Balances = await unipigWallet.getBalances(newAddress)
      balances.should.deep.equal({
        [UNI_TOKEN_TYPE]: 10,
        [PIGI_TOKEN_TYPE]: 10,
      })

      await unipigWallet.send(
        UNI_TOKEN_TYPE,
        newAddress,
        testRecipientAddress,
        10
      )

      const recipientBalances: Balances = await unipigWallet.getBalances(
        testRecipientAddress
      )
      recipientBalances[UNI_TOKEN_TYPE].should.equal(10)
    }).timeout(timeout)
  })
})
