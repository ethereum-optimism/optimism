import { WalletDB } from '../../../../src/services'
import { dbservice } from '../db.service'

export const walletdb = new WalletDB(dbservice)
