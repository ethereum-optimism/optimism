import { Contract } from '../Contract';
import { encodedNumber } from '../Encoding';
import { Callable, Sendable } from '../Invokation';

export interface ReservoirMethods {
  drip(): Sendable<void>;
  dripped(): Callable<number>;
  dripStart(): Callable<number>;
  dripRate(): Callable<number>;
  token(): Callable<string>;
  target(): Callable<string>;
}

export interface Reservoir extends Contract {
  methods: ReservoirMethods;
  name: string;
}
