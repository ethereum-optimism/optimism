"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toStrippedHexString = exports.toUnpaddedHexString = exports.toBytesN = exports.toBytes32 = exports.toUintN = exports.toUint8 = exports.toUint256 = exports.toHexString = exports.fromHexString = void 0;
const ethers_1 = require("ethers");
const fromHexString = (buf) => {
    if (typeof buf === 'string' && buf.startsWith('0x')) {
        return Buffer.from(buf.slice(2), 'hex');
    }
    return Buffer.from(buf);
};
exports.fromHexString = fromHexString;
const toHexString = (buf) => {
    if (typeof buf === 'number') {
        return ethers_1.BigNumber.from(buf).toHexString();
    }
    else {
        return '0x' + exports.fromHexString(buf).toString('hex');
    }
};
exports.toHexString = toHexString;
const toUint256 = (num) => {
    return exports.toUintN(num, 32);
};
exports.toUint256 = toUint256;
const toUint8 = (num) => {
    return exports.toUintN(num, 1);
};
exports.toUint8 = toUint8;
const toUintN = (num, n) => {
    return ('0x' +
        ethers_1.BigNumber.from(num)
            .toHexString()
            .slice(2)
            .padStart(n * 2, '0'));
};
exports.toUintN = toUintN;
const toBytes32 = (buf) => {
    return exports.toBytesN(buf, 32);
};
exports.toBytes32 = toBytes32;
const toBytesN = (buf, n) => {
    return ('0x' +
        exports.toHexString(buf)
            .slice(2)
            .padStart(n * 2, '0'));
};
exports.toBytesN = toBytesN;
const toUnpaddedHexString = (buf) => {
    const hex = '0x' + exports.toHexString(buf).slice(2).replace(/^0+/, '');
    if (hex === '0x') {
        return '0x0';
    }
    else {
        return hex;
    }
};
exports.toUnpaddedHexString = toUnpaddedHexString;
const toStrippedHexString = (buf) => {
    const hex = exports.toUnpaddedHexString(buf).slice(2);
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
//# sourceMappingURL=hex-utils.js.map