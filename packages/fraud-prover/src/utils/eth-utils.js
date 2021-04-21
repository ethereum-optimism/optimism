"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeAccountState = exports.encodeAccountState = void 0;
const ethers_1 = require("ethers");
const ethereumjs_util_1 = require("ethereumjs-util");
const hex_utils_1 = require("./hex-utils");
const encodeAccountState = (state) => {
    return new ethereumjs_util_1.Account(new ethereumjs_util_1.BN(state.nonce), new ethereumjs_util_1.BN(state.balance.toNumber()), hex_utils_1.fromHexString(state.storageRoot), hex_utils_1.fromHexString(state.codeHash)).serialize();
};
exports.encodeAccountState = encodeAccountState;
const decodeAccountState = (state) => {
    const account = ethereumjs_util_1.Account.fromRlpSerializedAccount(state);
    return {
        nonce: account.nonce.toNumber(),
        balance: ethers_1.BigNumber.from(account.nonce.toNumber()),
        storageRoot: hex_utils_1.toHexString(account.stateRoot),
        codeHash: hex_utils_1.toHexString(account.codeHash),
    };
};
exports.decodeAccountState = decodeAccountState;
//# sourceMappingURL=eth-utils.js.map