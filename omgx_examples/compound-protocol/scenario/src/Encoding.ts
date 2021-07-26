import BigNumber from 'bignumber.js';

const smallEnoughNumber = new BigNumber('100000000');

export type encodedNumber = number | BigNumber;

// Returns the mantissa of an Exp with given floating value
export function getExpMantissa(float: number): encodedNumber {
  // Workaround from https://github.com/ethereum/web3.js/issues/1920
  const str = Math.floor(float * 1.0e18).toString();

  return toEncodableNum(str);
}

export function toEncodableNum(amountArgRaw: string | encodedNumber): encodedNumber {
  const bigNumber = new BigNumber(amountArgRaw);
  return bigNumber.lt(smallEnoughNumber) ? bigNumber.toNumber() : bigNumber;
}
