"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
require("./setup");
const abi_1 = require("../src/abi");
const verify_1 = require("../src/verify");
const constants_1 = require("./constants");
describe('Validation', () => {
    describe('validStateTransition', () => {
        it('should correctly accept a valid state transition', async () => {
            const preimage = '0x' + Buffer.from('hello').toString('hex');
            const hash = '0x1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8';
            const encodedData = abi_1.encodeParams(['bytes32'], [hash]);
            const oldState = abi_1.encodeParams(['bytes'], [encodedData]);
            const newState = abi_1.encodeParams(['bytes'], ['0x00']);
            const witness = abi_1.encodeParams(['bytes'], [preimage]);
            const valid = await verify_1.validStateTransition(oldState, newState, witness, constants_1.PREIMAGE_BYTECODE);
            valid.should.be.true;
        });
        it('should correctly reject an invalid state transition', async () => {
            const preimage = '0x' + Buffer.from('goodbye').toString('hex');
            const hash = '0x1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8';
            const encodedData = abi_1.encodeParams(['bytes32'], [hash]);
            const oldState = abi_1.encodeParams(['bytes'], [encodedData]);
            const newState = abi_1.encodeParams(['bytes'], ['0x00']);
            const witness = abi_1.encodeParams(['bytes'], [preimage]);
            const valid = await verify_1.validStateTransition(oldState, newState, witness, constants_1.PREIMAGE_BYTECODE);
            valid.should.be.false;
        });
    });
});
