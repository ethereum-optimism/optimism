"use strict";
exports.__esModule = true;
exports.decodeAccountState = exports.encodeAccountState = void 0;
var ethers_1 = require("ethers");
var ethereumjs_util_1 = require("ethereumjs-util");
var core_utils_1 = require("@eth-optimism/core-utils");
var encodeAccountState = function (state) {
    return new ethereumjs_util_1.Account(new ethereumjs_util_1.BN(state.nonce), new ethereumjs_util_1.BN(state.balance.toNumber()), core_utils_1.fromHexString(state.storageRoot), core_utils_1.fromHexString(state.codeHash)).serialize();
};
exports.encodeAccountState = encodeAccountState;
var decodeAccountState = function (state) {
    var account = ethereumjs_util_1.Account.fromRlpSerializedAccount(state);
    return {
        nonce: account.nonce.toNumber(),
        balance: ethers_1.BigNumber.from(account.nonce.toNumber()),
        storageRoot: core_utils_1.toHexString(account.stateRoot),
        codeHash: core_utils_1.toHexString(account.codeHash)
    };
};
exports.decodeAccountState = decodeAccountState;
