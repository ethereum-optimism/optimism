import {World} from './World';
import {Map} from 'immutable';

export const accountMap = {
  "default": 0,
  "root": 0,
  "admin": 0,
  "first": 0,

  "bank": 1,
  "second": 1,

  "geoff": 2,
  "third": 2,
  "guardian": 2,

  "torrey": 3,
  "fourth": 3,

  "robert": 4,
  "fifth": 4,

  "coburn": 5,
  "sixth": 5,

  "jared": 6,
  "seventh": 6
};

export interface Account {
  name: string
  address: string
}

export type Accounts = Map<string, Account>

export function accountAliases(index: number): string[] {
  return Object.entries(accountMap).filter(([k,v]) => v === index).map(([k,v]) => k);
}

export function loadAccounts(accounts: string[]): Accounts {
  return Object.entries(accountMap).reduce((acc, [name, index]) => {
    if (accounts[index]) {
      return acc.set(name, { name: name, address: accounts[index] });
    } else {
      return acc;
    }
  }, <Map<string, Account>>Map({}));
}
