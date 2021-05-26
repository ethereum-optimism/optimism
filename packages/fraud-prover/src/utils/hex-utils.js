"use strict";
exports.__esModule = true;
exports.toUintN = exports.toUint8 = exports.toUint256 = exports.toBytesN = exports.toBytes32 = exports.toStrippedHexString = exports.toUnpaddedHexString = void 0;
var ethers_1 = require("ethers");
var core_utils_1 = require("@eth-optimism/core-utils");
var toUnpaddedHexString = function (buf) {
    // prettier-ignore
    var hex = '0x' +
        core_utils_1.toHexString(buf)
            .slice(2)
            .replace(/^0+/, '');
    if (hex === '0x') {
        return '0x0';
    }
    else {
        return hex;
    }
};
exports.toUnpaddedHexString = toUnpaddedHexString;
var toStrippedHexString = function (buf) {
    var hex = exports.toUnpaddedHexString(buf).slice(2);
    if (hex === '0') {
        return '0x';
    }
    else if (hex.length % 2 === 1) {
        return '0x' + '0' + hex;
    }
    else {
        return '0x' + hex;
    }
};
exports.toStrippedHexString = toStrippedHexString;
var toBytes32 = function (buf) {
    return exports.toBytesN(buf, 32);
};
exports.toBytes32 = toBytes32;
var toBytesN = function (buf, n) {
    return ('0x' +
        core_utils_1.toHexString(buf)
            .slice(2)
            .padStart(n * 2, '0'));
};
exports.toBytesN = toBytesN;
var toUint256 = function (num) {
    return exports.toUintN(num, 32);
};
exports.toUint256 = toUint256;
var toUint8 = function (num) {
    return exports.toUintN(num, 1);
};
exports.toUint8 = toUint8;
var toUintN = function (num, n) {
    return ('0x' +
        ethers_1.BigNumber.from(num)
            .toHexString()
            .slice(2)
            .padStart(n * 2, '0'));
};
exports.toUintN = toUintN;
/*

already in @eth-optimism/core-utils

export const fromHexString = (buf: Buffer | string): Buffer => {
  if (typeof buf === 'string' && buf.startsWith('0x')) {
    return Buffer.from(buf.slice(2), 'hex')
  }

  return Buffer.from(buf)
}

export const toHexString = (buf: Buffer | string | number | null): string => {
  if (typeof buf === 'number') {
    return BigNumber.from(buf).toHexString()
  } else {
    return '0x' + fromHexString(buf).toString('hex')
  }
}
*/
