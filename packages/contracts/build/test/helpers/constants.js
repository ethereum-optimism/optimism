"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getStorageXOR = exports.STORAGE_XOR = exports.Helper_TestRunner_BYTELEN = exports.NUISANCE_GAS_COSTS = exports.VERIFIED_EMPTY_CONTRACT_HASH = exports.NON_ZERO_ADDRESS = exports.ZERO_ADDRESS = exports.NON_NULL_BYTES32 = exports.NULL_BYTES32 = exports.FORCE_INCLUSION_PERIOD_SECONDS = exports.RUN_OVM_TEST_GAS = exports.OVM_TX_GAS_LIMIT = exports.DEFAULT_ACCOUNTS_BUIDLER = exports.DEFAULT_ACCOUNTS = void 0;
const ethers_1 = require("ethers");
const ethereum_waffle_1 = require("ethereum-waffle");
const buffer_xor_1 = __importDefault(require("buffer-xor"));
const utils_1 = require("./utils");
exports.DEFAULT_ACCOUNTS = ethereum_waffle_1.defaultAccounts;
exports.DEFAULT_ACCOUNTS_BUIDLER = ethereum_waffle_1.defaultAccounts.map((account) => {
    return {
        balance: ethers_1.ethers.BigNumber.from(account.balance).toHexString(),
        privateKey: account.secretKey,
    };
});
exports.OVM_TX_GAS_LIMIT = 10000000;
exports.RUN_OVM_TEST_GAS = 20000000;
exports.FORCE_INCLUSION_PERIOD_SECONDS = 600;
exports.NULL_BYTES32 = utils_1.makeHexString('00', 32);
exports.NON_NULL_BYTES32 = utils_1.makeHexString('11', 32);
exports.ZERO_ADDRESS = utils_1.makeAddress('00');
exports.NON_ZERO_ADDRESS = utils_1.makeAddress('11');
exports.VERIFIED_EMPTY_CONTRACT_HASH = '0x00004B1DC0DE000000004B1DC0DE000000004B1DC0DE000000004B1DC0DE0000';
exports.NUISANCE_GAS_COSTS = {
    NUISANCE_GAS_SLOAD: 20000,
    NUISANCE_GAS_SSTORE: 20000,
    MIN_NUISANCE_GAS_PER_CONTRACT: 30000,
    NUISANCE_GAS_PER_CONTRACT_BYTE: 100,
    MIN_GAS_FOR_INVALID_STATE_ACCESS: 30000,
};
exports.Helper_TestRunner_BYTELEN = 3654;
exports.STORAGE_XOR = '0xfeedfacecafebeeffeedfacecafebeeffeedfacecafebeeffeedfacecafebeef';
exports.getStorageXOR = (key) => {
    return utils_1.toHexString(buffer_xor_1.default(utils_1.fromHexString(key), utils_1.fromHexString(exports.STORAGE_XOR)));
};
//# sourceMappingURL=constants.js.map