"use strict";
/* tslint:disable:no-any */
/* Credit: https://github.com/ethereum/remix */
Object.defineProperty(exports, "__esModule", { value: true });
const ethers_1 = require("ethers");
const extractSize = (type) => {
    const size = type.match(/([a-zA-Z0-9])(\[.*\])/);
    return size ? size[2] : '';
};
const makeFullTypeDefinition = (typeDef) => {
    if (typeDef && typeDef.type.indexOf('tuple') === 0 && typeDef.components) {
        const innerTypes = typeDef.components.map((innerType) => {
            return makeFullTypeDefinition(innerType);
        });
        return `tuple(${innerTypes.join(',')})${extractSize(typeDef.type)}`;
    }
    return typeDef.type;
};
exports.encodeParams = (types, args) => {
    const abiCoder = new ethers_1.ethers.utils.AbiCoder();
    return abiCoder.encode(types, args);
};
const encodeMethodParams = (methodAbi, args) => {
    const types = [];
    if (methodAbi.inputs && methodAbi.inputs.length) {
        for (const input of methodAbi.inputs) {
            const type = input.type;
            types.push(type.indexOf('tuple') === 0 ? makeFullTypeDefinition(input) : type);
            if (args.length < types.length) {
                args.push('');
            }
        }
    }
    return exports.encodeParams(types, args);
};
const encodeMethodId = (methodAbi) => {
    if (methodAbi.type === 'fallback') {
        return '0x';
    }
    const abi = new ethers_1.ethers.utils.Interface([methodAbi]);
    const fn = abi.functions[methodAbi.name];
    return fn.sighash;
};
exports.encodeMethod = (methodAbi, args) => {
    const encodedParams = encodeMethodParams(methodAbi, args).replace('0x', '');
    const methodId = encodeMethodId(methodAbi);
    return methodId + encodedParams;
};
exports.decodeResponse = (methodAbi, response) => {
    if (!methodAbi.outputs || methodAbi.outputs.length === 0) {
        return {};
    }
    const outputTypes = [];
    for (const output of methodAbi.outputs) {
        const type = output.type;
        outputTypes.push(type.indexOf('tuple') === 0 ? makeFullTypeDefinition(output) : type);
    }
    if (!response.length) {
        response = new Uint8Array(32 * methodAbi.outputs.length);
    }
    const abiCoder = new ethers_1.ethers.utils.AbiCoder();
    return abiCoder.decode(outputTypes, response);
};
