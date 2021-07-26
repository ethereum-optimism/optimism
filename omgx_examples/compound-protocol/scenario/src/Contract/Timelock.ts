import { Contract } from '../Contract';
import { Callable, Sendable } from '../Invokation';
import { encodedNumber } from '../Encoding';

interface TimelockMethods {
  admin(): Callable<string>;
  pendingAdmin(): Callable<string>;
  delay(): Callable<number>;
  queuedTransactions(txHash: string): Callable<boolean>;
  setDelay(delay: encodedNumber): Sendable<void>;
  acceptAdmin(): Sendable<void>;
  setPendingAdmin(admin: string): Sendable<void>;
  queueTransaction(
    target: string,
    value: encodedNumber,
    signature: string,
    data: string,
    eta: encodedNumber
  ): Sendable<string>;
  cancelTransaction(
    target: string,
    value: encodedNumber,
    signature: string,
    data: string,
    eta: encodedNumber
  ): Sendable<void>;
  executeTransaction(
    target: string,
    value: encodedNumber,
    signature: string,
    data: string,
    eta: encodedNumber
  ): Sendable<string>;

  blockTimestamp(): Callable<number>;
  harnessFastForward(seconds: encodedNumber): Sendable<void>;
  harnessSetBlockTimestamp(seconds: encodedNumber): Sendable<void>;
  harnessSetAdmin(admin: string): Sendable<void>;
}

export interface Timelock extends Contract {
  methods: TimelockMethods;
}
