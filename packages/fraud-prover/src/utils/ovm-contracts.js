"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.loadProxyFromManager = exports.loadContractFromManager = exports.loadContract = void 0;
const ethers_1 = require("ethers");
const contract_defs_1 = require("@eth-optimism/contracts/build/src/contract-defs");
const constants_1 = require("./constants");
const loadContract = (name, address, provider) => {
    return new ethers_1.Contract(address, contract_defs_1.getContractInterface(name), provider);
};
exports.loadContract = loadContract;
const loadContractFromManager = async (name, Lib_AddressManager, provider) => {
    const address = await Lib_AddressManager.getAddress(name);
    if (address === constants_1.ZERO_ADDRESS) {
        throw new Error(`Lib_AddressManager does not have a record for a contract named: ${name}`);
    }
    return exports.loadContract(name, address, provider);
};
exports.loadContractFromManager = loadContractFromManager;
const loadProxyFromManager = async (name, proxy, Lib_AddressManager, provider) => {
    const address = await Lib_AddressManager.getAddress(proxy);
    if (address === constants_1.ZERO_ADDRESS) {
        throw new Error(`Lib_AddressManager does not have a record for a contract named: ${proxy}`);
    }
    return exports.loadContract(name, address, provider);
};
exports.loadProxyFromManager = loadProxyFromManager;
//# sourceMappingURL=ovm-contracts.js.map