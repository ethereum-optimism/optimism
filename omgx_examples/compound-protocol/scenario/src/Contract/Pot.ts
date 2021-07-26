import { Contract } from '../Contract';
import { Callable, Sendable } from '../Invokation';
import { encodedNumber } from '../Encoding';

interface PotMethods {
  chi(): Callable<number>;
  dsr(): Callable<number>;
  rho(): Callable<number>;
  pie(address: string): Callable<number>;
  drip(): Sendable<void>;
  file(what: string, data: encodedNumber): Sendable<void>;
  join(amount: encodedNumber): Sendable<void>;
  exit(amount: encodedNumber): Sendable<void>;
}

export interface Pot extends Contract {
  methods: PotMethods;
  name: string;
}
