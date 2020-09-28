"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const config_1 = require("@nomiclabs/buidler/config");
const constants_1 = require("./test/helpers/constants");
config_1.usePlugin('@nomiclabs/buidler-ethers');
config_1.usePlugin('@nomiclabs/buidler-waffle');
require("./test/helpers/buidler/modify-compiler");
const config = {
    networks: {
        buidlerevm: {
            accounts: constants_1.DEFAULT_ACCOUNTS_BUIDLER,
            blockGasLimit: constants_1.RUN_OVM_TEST_GAS * 2,
        },
    },
    mocha: {
        timeout: 50000,
    },
    solc: {
        version: '0.7.0',
        optimizer: { enabled: true, runs: 200 },
    },
};
exports.default = config;
//# sourceMappingURL=buidler.config.js.map