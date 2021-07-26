import {Contract} from '../Contract';
import {Callable, Sendable} from '../Invokation';
import {encodedNumber} from '../Encoding';

interface Erc20Methods {
  name(): Callable<string>
  symbol(): Callable<string>
  decimals(): Callable<string>
  totalSupply(): Callable<number>
  balanceOf(string): Callable<string>
  allowance(owner: string, spender: string): Callable<string>
  approve(address: string, amount: encodedNumber): Sendable<number>
  allocateTo(address: string, amount: encodedNumber): Sendable<number>
  transfer(address: string, amount: encodedNumber): Sendable<boolean>
  transferFrom(owner: string, spender: string, amount: encodedNumber): Sendable<boolean>
  setFail(fail: boolean): Sendable<void>
  pause(): Sendable<void>
  unpause(): Sendable<void>
  setParams(newBasisPoints: encodedNumber, maxFee: encodedNumber): Sendable<void>
}

export interface Erc20 extends Contract {
  methods: Erc20Methods
  name: string
}
