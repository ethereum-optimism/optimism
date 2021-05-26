"use strict";
exports.__esModule = true;
exports.hashOvmTransaction = exports.encodeOvmTransaction = void 0;
var ethers_1 = require("ethers");
var hex_utils_1 = require("./hex-utils");
var core_utils_1 = require("@eth-optimism/core-utils");
var encodeOvmTransaction = function (transaction) {
    return core_utils_1.toHexString(Buffer.concat([
        core_utils_1.fromHexString(hex_utils_1.toUint256(transaction.timestamp)),
        core_utils_1.fromHexString(hex_utils_1.toUint256(transaction.blockNumber)),
        core_utils_1.fromHexString(hex_utils_1.toUint8(transaction.l1QueueOrigin)),
        core_utils_1.fromHexString(transaction.l1TxOrigin),
        core_utils_1.fromHexString(transaction.entrypoint),
        core_utils_1.fromHexString(hex_utils_1.toUint256(transaction.gasLimit)),
        core_utils_1.fromHexString(transaction.data),
    ]));
};
exports.encodeOvmTransaction = encodeOvmTransaction;
var hashOvmTransaction = function (transaction) {
    return ethers_1.ethers.utils.keccak256(exports.encodeOvmTransaction(transaction));
};
exports.hashOvmTransaction = hashOvmTransaction;
