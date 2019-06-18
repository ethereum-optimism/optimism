import '../../setup'

/* External Imports */
import MemDown from 'memdown'

/* Internal Imports */
import { Keystore } from '../../../src/interfaces'
import { BaseDB } from '../../../src/app/common/db'
import { DefaultWalletDB } from '../../../src/app/client/wallet-db'

const keystore: Keystore = {
  address: '2600a448db443dc49f3c0b6bf46e6f9110914568',
  id: '712b2934-7ccd-4ef7-87f3-6384627d5b7d',
  version: 3,
  crypto: {
    cipher: 'aes-128-cbc',
    ciphertext:
      '225c3c42c2d7834c844a26070b13da6d5ac0e812022e4a4be434833aef430ae6',
    cipherparams: {
      iv: '75304b13fcf01c67536eb985f88dfc43',
    },
    kdf: 'scrypt',
    kdfparams: {
      dklen: 32,
      n: 262144,
      p: 1,
      r: 8,
      salt: 'cd623230c41b3c8a8a88547e150da6ca1653bff04951cedd13791243d910cb21',
    },
    mac: 'cbe1a233297b97518efdbebe4a250bf5b29461537384f0d83d9f016c747eff5f',
  },
}

describe('DefaultWalletDB', () => {
  let walletdb: DefaultWalletDB
  beforeEach(() => {
    walletdb = new DefaultWalletDB(new BaseDB(new MemDown('') as any))
  })

  describe('putKeystore', () => {
    it('should correctly insert valid keystore files', async () => {
      await walletdb.putKeystore(keystore).should.be.fulfilled
    })
  })

  describe('getKeystore', () => {
    it('should correctly query a keystore file', async () => {
      await walletdb.putKeystore(keystore)

      const stored = await walletdb.getKeystore(keystore.address)
      stored.should.deep.equal(keystore)
    })

    it('should throw if the keystore file does not exist', async () => {
      await walletdb
        .getKeystore(keystore.address)
        .should.be.rejectedWith('Keystore file does not exist.')
    })
  })

  describe('listAccounts', () => {
    it('should return a single address if only one keystore', async () => {
      await walletdb.putKeystore(keystore)

      const addresses = await walletdb.listAccounts()
      addresses.should.deep.equal([keystore.address])
    })
  })
})
