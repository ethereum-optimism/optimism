"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hashOvmTransaction = exports.encodeOvmTransaction = void 0;
const ethers_1 = require("ethers");
const hex_utils_1 = require("./hex-utils");
const encodeOvmTransaction = (transaction) => {
    return hex_utils_1.toHexString(Buffer.concat([
        hex_utils_1.fromHexString(hex_utils_1.toUint256(transaction.timestamp)),
        hex_utils_1.fromHexString(hex_utils_1.toUint256(transaction.blockNumber)),
        hex_utils_1.fromHexString(hex_utils_1.toUint8(transaction.l1QueueOrigin)),
        hex_utils_1.fromHexString(transaction.l1TxOrigin),
        hex_utils_1.fromHexString(transaction.entrypoint),
        hex_utils_1.fromHexString(hex_utils_1.toUint256(transaction.gasLimit)),
        hex_utils_1.fromHexString(transaction.data),
    ]));
};
exports.encodeOvmTransaction = encodeOvmTransaction;
const hashOvmTransaction = (transaction) => {
    return ethers_1.ethers.utils.keccak256(exports.encodeOvmTransaction(transaction));
};
exports.hashOvmTransaction = hashOvmTransaction;
//# sourceMappingURL=ovm-utils.js.map