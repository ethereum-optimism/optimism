"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const ethereumjs_vm_1 = require("ethereumjs-vm");
class VM {
    constructor(options) {
        this.vm = new ethereumjs_vm_1.default(options);
    }
    generateGenesis(initState) {
        return new Promise((resolve, reject) => {
            this.vm.stateManager.generateGenesis(initState, (err, result) => {
                if (err) {
                    reject(err);
                }
                resolve(result);
            });
        });
    }
    runTx(options) {
        return new Promise((resolve, reject) => {
            this.vm.runTx(options, (err, result) => {
                if (err) {
                    reject(err);
                }
                resolve(result);
            });
        });
    }
}
exports.VM = VM;
