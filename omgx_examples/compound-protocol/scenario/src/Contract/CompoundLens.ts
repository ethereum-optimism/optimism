import { Contract } from '../Contract';
import { encodedNumber } from '../Encoding';
import { Callable, Sendable } from '../Invokation';

export interface CompoundLensMethods {
  cTokenBalances(cToken: string, account: string): Sendable<[string,number,number,number,number,number]>;
  cTokenBalancesAll(cTokens: string[], account: string): Sendable<[string,number,number,number,number,number][]>;
  cTokenMetadata(cToken: string): Sendable<[string,number,number,number,number,number,number,number,number,boolean,number,string,number,number]>;
  cTokenMetadataAll(cTokens: string[]): Sendable<[string,number,number,number,number,number,number,number,number,boolean,number,string,number,number][]>;
  cTokenUnderlyingPrice(cToken: string): Sendable<[string,number]>;
  cTokenUnderlyingPriceAll(cTokens: string[]): Sendable<[string,number][]>;
  getAccountLimits(comptroller: string, account: string): Sendable<[string[],number,number]>;
}

export interface CompoundLens extends Contract {
  methods: CompoundLensMethods;
  name: string;
}
