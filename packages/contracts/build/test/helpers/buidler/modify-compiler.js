"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const fs_extra_1 = __importDefault(require("fs-extra"));
const config_1 = require("@nomiclabs/buidler/config");
const strings_1 = require("@nomiclabs/buidler/internal/util/strings");
const artifacts_1 = require("@nomiclabs/buidler/internal/artifacts");
const task_names_1 = require("@nomiclabs/buidler/builtin-tasks/task-names");
config_1.internalTask(task_names_1.TASK_COMPILE_GET_COMPILER_INPUT, async (_, { config, run }, runSuper) => {
    const input = await runSuper();
    input.settings.outputSelection['*']['*'].push('storageLayout');
    return input;
});
config_1.internalTask(task_names_1.TASK_BUILD_ARTIFACTS).setAction(async ({ force }, { config, run }) => {
    const sources = await run(task_names_1.TASK_COMPILE_GET_SOURCE_PATHS);
    if (sources.length === 0) {
        console.log('No Solidity source file available.');
        return;
    }
    const isCached = await run(task_names_1.TASK_COMPILE_CHECK_CACHE, { force });
    if (isCached) {
        console.log('All contracts have already been compiled, skipping compilation.');
        return;
    }
    const compilationOutput = await run(task_names_1.TASK_COMPILE_COMPILE);
    if (compilationOutput === undefined) {
        return;
    }
    await fs_extra_1.default.ensureDir(config.paths.artifacts);
    let numberOfContracts = 0;
    for (const file of Object.values(compilationOutput.contracts)) {
        for (const [contractName, contractOutput] of Object.entries(file)) {
            const artifact = artifacts_1.getArtifactFromContractOutput(contractName, contractOutput);
            numberOfContracts += 1;
            artifact.storageLayout = contractOutput.storageLayout;
            await artifacts_1.saveArtifact(config.paths.artifacts, artifact);
        }
    }
    console.log('Compiled', numberOfContracts, strings_1.pluralize(numberOfContracts, 'contract'), 'successfully');
});
//# sourceMappingURL=modify-compiler.js.map