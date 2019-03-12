import { should } from '../../../setup'

/* External Imports */
import BigNum from 'bn.js'

/* Internal Imports */
import { StateManager } from '../../../../src/services/chain/state-manager'
import { Deposit, Exit, Snapshot } from '../../../../src/models/chain'

const accounts = constants.ACCOUNTS
const models = serialization.models
const UnsignedTransaction = models.UnsignedTransaction

describe('SnapshotManager', () => {
  const deposit = new Deposit({
    token: new BigNum(0),
    start: new BigNum(0),
    end: new BigNum(100),
    block: new BigNum(0),
    owner: accounts[0].address,
  })

  let snapshotManager: SnapshotManager
  beforeEach(() => {
    snapshotManager = new SnapshotManager()
  })

  describe('applyDeposit', () => {
    it('should be able to apply a deposit', () => {
      snapshotManager.applyDeposit(deposit)

      snapshotManager.equals([Snapshot.from(deposit)]).should.be.true
    })

    it('should not apply a deposit with start greater than end', () => {
      const badDeposit = new Deposit({
        token: new BigNum(0),
        start: new BigNum(100),
        end: new BigNum(0),
        block: new BigNum(0),
        owner: accounts[0].address,
      })

      should.Throw(() => {
        snapshotManager.applyDeposit(badDeposit)
      }, 'Invalid snapshot')
    })
  })

  describe('applyTransaction', () => {
    it('should be able to apply a valid transaction', () => {
      const transaction = new UnsignedTransaction({
        block: 1,
        transfers: [
          {
            start: 0,
            end: 100,
            sender: accounts[0].address,
            recipient: accounts[1].address,
          },
        ],
      })
      const expected = new Snapshot({
        start: new BigNum(0),
        end: new BigNum(100),
        block: new BigNum(1),
        owner: accounts[1].address,
      })

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyTransaction(transaction)

      snapshotManager.equals([expected]).should.be.true
    })

    it('should apply a transaction that goes over an existing range', () => {
      const transaction = new UnsignedTransaction({
        block: 1,
        transfers: [
          {
            start: 0,
            end: 200,
            sender: accounts[0].address,
            recipient: accounts[1].address,
          },
        ],
      })
      const expected = new Snapshot({
        start: new BigNum(0),
        end: new BigNum(100),
        block: new BigNum(1),
        owner: accounts[1].address,
      })

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyTransaction(transaction)

      snapshotManager.equals([expected]).should.be.true
    })

    it('should apply a transaction that goes under an existing range', () => {
      const transaction = new UnsignedTransaction({
        block: 1,
        transfers: [
          {
            start: 0,
            end: 50,
            sender: accounts[0].address,
            recipient: accounts[1].address,
          },
        ],
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(50),
          block: new BigNum(1),
          owner: accounts[1].address,
        }),
        new Snapshot({
          start: new BigNum(50),
          end: new BigNum(100),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyTransaction(transaction)

      snapshotManager.equals(expected).should.be.true
    })

    it('should apply a transaction with implicit start and ends', () => {
      const transaction = new UnsignedTransaction({
        block: 1,
        transfers: [
          {
            implicitStart: 0,
            start: 25,
            end: 75,
            implicitEnd: 100,
            sender: accounts[0].address,
            recipient: accounts[1].address,
          },
        ],
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(25),
          block: new BigNum(1),
          owner: accounts[0].address,
        }),
        new Snapshot({
          start: new BigNum(25),
          end: new BigNum(75),
          block: new BigNum(1),
          owner: accounts[1].address,
        }),
        new Snapshot({
          start: new BigNum(75),
          end: new BigNum(100),
          block: new BigNum(1),
          owner: accounts[0].address,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyTransaction(transaction)

      snapshotManager.equals(expected).should.be.true
    })

    it('should apply a transaction with only an implicit end', () => {
      const transaction = new UnsignedTransaction({
        block: 1,
        transfers: [
          {
            start: 0,
            end: 75,
            implicitEnd: 100,
            sender: accounts[0].address,
            recipient: accounts[1].address,
          },
        ],
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(75),
          block: new BigNum(1),
          owner: accounts[1].address,
        }),
        new Snapshot({
          start: new BigNum(75),
          end: new BigNum(100),
          block: new BigNum(1),
          owner: accounts[0].address,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyTransaction(transaction)

      snapshotManager.equals(expected).should.be.true
    })

    it('should apply a transaction where only an implicit part overlaps', () => {
      const deposit2 = new Deposit({
        token: new BigNum(0),
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
        owner: accounts[1].address,
      })
      const transaction = new UnsignedTransaction({
        block: 1,
        transfers: [
          {
            implicitStart: 0,
            start: 100,
            end: 200,
            sender: accounts[1].address,
            recipient: accounts[2].address,
          },
        ],
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(100),
          block: new BigNum(1),
          owner: accounts[0].address,
        }),
        new Snapshot({
          start: new BigNum(100),
          end: new BigNum(200),
          block: new BigNum(1),
          owner: accounts[2].address,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyDeposit(deposit2)
      snapshotManager.applyTransaction(transaction)

      snapshotManager.equals(expected).should.be.true
    })
  })

  describe('validateTransaction', () => {
    it('should not verify a transaction with an invalid sender', () => {
      const transaction = new UnsignedTransaction({
        block: 1,
        transfers: [
          {
            start: 0,
            end: 100,
            sender: accounts[1].address,
            recipient: accounts[0].address,
          },
        ],
      })

      snapshotManager.applyDeposit(deposit)

      snapshotManager.validateTransaction(transaction).should.be.false
    })
  })

  describe('applyExit', () => {
    it('should be able to apply an exit that equals a range', () => {
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(0),
        end: new BigNum(100),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(100),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit with the same start but lower end than a range', () => {
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(0),
        end: new BigNum(75),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(75),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
        new Snapshot({
          start: new BigNum(75),
          end: new BigNum(100),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit with the same start higher lower end than a range', () => {
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(0),
        end: new BigNum(125),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(125),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit with the same end but lower start than a range', () => {
      const deposit2 = new Deposit({
        token: new BigNum(0),
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
        owner: accounts[0].address,
      })
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(50),
        end: new BigNum(200),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(50),
          end: new BigNum(200),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit2)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit with the same end but higher start than a range', () => {
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(25),
        end: new BigNum(100),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(25),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
        new Snapshot({
          start: new BigNum(25),
          end: new BigNum(100),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit with a lower start and lower end than a range', () => {
      const deposit2 = new Deposit({
        token: new BigNum(0),
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
        owner: accounts[0].address,
      })
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(50),
        end: new BigNum(150),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(50),
          end: new BigNum(150),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
        new Snapshot({
          start: new BigNum(150),
          end: new BigNum(200),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
      ]

      snapshotManager.applyDeposit(deposit2)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit with a higher start and higher end than a range', () => {
      const deposit2 = new Deposit({
        token: new BigNum(0),
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
        owner: accounts[0].address,
      })
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(150),
        end: new BigNum(250),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(100),
          end: new BigNum(150),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
        new Snapshot({
          start: new BigNum(150),
          end: new BigNum(250),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit2)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit with higher start and lower end than a range', () => {
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(25),
        end: new BigNum(75),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(25),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
        new Snapshot({
          start: new BigNum(75),
          end: new BigNum(100),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
        new Snapshot({
          start: new BigNum(25),
          end: new BigNum(75),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit that completely overlaps a range', () => {
      const deposit2 = new Deposit({
        token: new BigNum(0),
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
        owner: accounts[0].address,
      })
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(50),
        end: new BigNum(250),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(50),
          end: new BigNum(250),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit2)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit that overlaps two ranges', () => {
      const deposit2 = new Deposit({
        token: new BigNum(0),
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
        owner: accounts[0].address,
      })
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(0),
        end: new BigNum(200),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(200),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyDeposit(deposit2)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })

    it('should be able to apply an exit that partially overlaps two ranges', () => {
      const deposit2 = new Deposit({
        token: new BigNum(0),
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
        owner: accounts[0].address,
      })
      const exit = new Exit({
        token: new BigNum(0),
        block: new BigNum(1),
        start: new BigNum(50),
        end: new BigNum(150),
        id: new BigNum(0),
        owner: accounts[0].address,
      })
      const expected = [
        new Snapshot({
          start: new BigNum(0),
          end: new BigNum(50),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
        new Snapshot({
          start: new BigNum(150),
          end: new BigNum(200),
          block: new BigNum(0),
          owner: accounts[0].address,
        }),
        new Snapshot({
          start: new BigNum(50),
          end: new BigNum(150),
          block: new BigNum(1),
          owner: constants.NULL_ADDRESS,
        }),
      ]

      snapshotManager.applyDeposit(deposit)
      snapshotManager.applyDeposit(deposit2)
      snapshotManager.applyExit(exit)

      snapshotManager.equals(expected).should.be.true
    })
  })
})
