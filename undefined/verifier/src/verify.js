"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const bn_js_1 = require("bn.js");
const ethereumjs_tx_1 = require("ethereumjs-tx");
const abi_1 = require("./abi");
const constants_1 = require("./constants");
const vm_1 = require("./vm");
const vm = new vm_1.VM({ enableHomestead: true, activatePrecompiles: true });
let nonce = new bn_js_1.default(0);
const initGenesis = async () => {
    const genesisData = {
        [constants_1.ACCOUNT.address]: '1606938044258990275541962092341162602522202993782792835301376',
    };
    await vm.generateGenesis(genesisData);
};
const createContract = async (bytecode) => {
    const contractCreationTx = new ethereumjs_tx_1.default({
        data: bytecode,
        from: constants_1.ACCOUNT.address,
        gasLimit: '0xffffffff',
        gasPrice: '0x01',
        nonce: '0x' + nonce.toString('hex'),
        value: '0x00',
    });
    contractCreationTx.sign(constants_1.ACCOUNT.privateKey);
    const contractCreationTxResult = await vm.runTx({
        skipBalance: true,
        skipNonce: true,
        tx: contractCreationTx,
    });
    if (contractCreationTxResult.createdAddress === undefined) {
        throw new Error('Could not create contract.');
    }
    nonce = nonce.addn(1);
    return '0x' + contractCreationTxResult.createdAddress.toString('hex');
};
const getContractMethod = (name) => {
    const method = constants_1.PREDICATE_ABI.find((item) => {
        return item.name === name;
    });
    if (method === undefined) {
        throw new Error('Could not find method name.');
    }
    return method;
};
const callContractMethod = async (address, method, inputs) => {
    const methodAbi = getContractMethod(method);
    const methodData = abi_1.encodeMethod(methodAbi, inputs);
    const validationCallTx = new ethereumjs_tx_1.default({
        data: methodData,
        gasLimit: '0xffffffff',
        gasPrice: '0x01',
        nonce: '0x01',
        to: address,
        value: '0x00',
    });
    validationCallTx.sign(constants_1.ACCOUNT.privateKey);
    const result = await vm.runTx({
        skipBalance: true,
        skipNonce: true,
        tx: validationCallTx,
    });
    if (result.vm.exception === 0) {
        const error = result.vm.exceptionError;
        throw new Error(`${error.errorType}: ${error.error}`);
    }
    nonce = nonce.addn(1);
    const decoded = abi_1.decodeResponse(methodAbi, result.vm.return);
    return decoded;
};
exports.validStateTransition = async (oldState, newState, witness, bytecode) => {
    await initGenesis();
    const contractAddress = await createContract(bytecode);
    const result = await callContractMethod(contractAddress, 'validStateTransition', [oldState, newState, witness]);
    return result[0];
};
