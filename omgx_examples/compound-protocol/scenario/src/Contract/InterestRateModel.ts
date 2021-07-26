import {Contract} from '../Contract';
import {Callable, Sendable} from '../Invokation';
import {encodedNumber} from '../Encoding';

interface InterestRateModelMethods {
  getBorrowRate(cash: encodedNumber, borrows: encodedNumber, reserves: encodedNumber): Callable<number>
}

export interface InterestRateModel extends Contract {
  methods: InterestRateModelMethods
  name: string
}
