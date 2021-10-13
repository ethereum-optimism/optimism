import { classify } from '../scripts/classifiers'
import {
  Account,
  AccountType,
  StateDump,
  SurgeryDataSources,
} from '../scripts/types'

export const findAccountsByType = (
  dump: StateDump,
  data: SurgeryDataSources,
  type: AccountType
): Account[] => {
  return dump.filter((account) => {
    return classify(account, data) === type
  })
}
