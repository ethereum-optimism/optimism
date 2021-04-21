"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.L2ProviderWrapper = void 0;
const hex_utils_1 = require("../hex-utils");
class L2ProviderWrapper {
    constructor(provider) {
        this.provider = provider;
    }
    async getStateRoot(index) {
        const block = await this.provider.send('eth_getBlockByNumber', [
            hex_utils_1.toUnpaddedHexString(index),
            false,
        ]);
        return block.stateRoot;
    }
    async getTransaction(index) {
        const transaction = await this.provider.send('eth_getTransactionByBlockNumberAndIndex', [hex_utils_1.toUnpaddedHexString(index), '0x0']);
        return transaction.input;
    }
    async getProof(index, address, slots = []) {
        return this.provider.send('eth_getProof', [
            address,
            slots,
            hex_utils_1.toUnpaddedHexString(index),
        ]);
    }
    async getStateDiffProof(index) {
        const proof = await this.provider.send('eth_getStateDiffProof', [
            hex_utils_1.toUnpaddedHexString(index),
        ]);
        return {
            header: proof.header,
            accountStateProofs: proof.accounts,
        };
    }
    async getRollupInfo() {
        return this.provider.send('rollup_getInfo', []);
    }
    async getAddressManagerAddress() {
        const rollupInfo = await this.getRollupInfo();
        return rollupInfo.addresses.addressResolver;
    }
}
exports.L2ProviderWrapper = L2ProviderWrapper;
//# sourceMappingURL=l2-provider-wrapper.js.map