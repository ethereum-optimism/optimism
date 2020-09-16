/* External Imports */
import fsExtra from 'fs-extra'
import { internalTask } from '@nomiclabs/buidler/config'
import { pluralize } from '@nomiclabs/buidler/internal/util/strings'
import { getArtifactFromContractOutput, saveArtifact } from '@nomiclabs/buidler/internal/artifacts'
import {
  TASK_COMPILE_GET_COMPILER_INPUT,
  TASK_BUILD_ARTIFACTS,
  TASK_COMPILE_GET_SOURCE_PATHS,
  TASK_COMPILE_CHECK_CACHE,
  TASK_COMPILE_COMPILE
} from '@nomiclabs/buidler/builtin-tasks/task-names'

internalTask(
  TASK_COMPILE_GET_COMPILER_INPUT,
  async (_, { config, run }, runSuper) => {
    const input = await runSuper();

    // Insert the "storageLayout" input option.
    input.settings.outputSelection['*']['*'].push('storageLayout')

    return input
  }
)

internalTask(TASK_BUILD_ARTIFACTS).setAction(
  async ({ force }, { config, run }) => {
    const sources = await run(TASK_COMPILE_GET_SOURCE_PATHS);

    if (sources.length === 0) {
      console.log("No Solidity source file available.");
      return;
    }
  
    const isCached: boolean = await run(TASK_COMPILE_CHECK_CACHE, { force });
  
    if (isCached) {
      console.log(
        "All contracts have already been compiled, skipping compilation."
      );
      return;
    }
  
    const compilationOutput = await run(TASK_COMPILE_COMPILE);
  
    if (compilationOutput === undefined) {
      return;
    }
  
    await fsExtra.ensureDir(config.paths.artifacts);
    let numberOfContracts = 0;
  
    for (const file of Object.values<any>(compilationOutput.contracts)) {
      for (const [contractName, contractOutput] of Object.entries(file)) {
        const artifact: any = getArtifactFromContractOutput(
          contractName,
          contractOutput
        );
        numberOfContracts += 1;

        // Only difference here, set the "storageLayout" field of the artifact.
        artifact.storageLayout = (contractOutput as any).storageLayout
  
        await saveArtifact(config.paths.artifacts, artifact);
      }
    }
  
    console.log(
      "Compiled",
      numberOfContracts,
      pluralize(numberOfContracts, "contract"),
      "successfully"
    );
  }
)
