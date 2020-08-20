import * as path from 'path'
import * as fs from 'fs'
import { internalTask } from '@nomiclabs/buidler/config'
import { SolcInput } from '@nomiclabs/buidler/types'
import { Compiler } from '@nomiclabs/buidler/internal/solidity/compiler'
import { TASK_COMPILE_RUN_COMPILER } from '@nomiclabs/buidler/builtin-tasks/task-names'

internalTask(TASK_COMPILE_RUN_COMPILER).setAction(
  async ({ input }: { input: SolcInput }, { config }) => {
    let customCompiler: any
    if (fs.existsSync((config as any).solc.path)) {
      customCompiler = require((config as any).solc.path)
    }

    const compiler = new Compiler(
      customCompiler ? customCompiler.version() : config.solc.version,
      path.join(config.paths.cache, 'compilers')
    )

    if (customCompiler) {
      compiler['getSolc' as any] = () => {
        return customCompiler
      }
    }

    return compiler.compile(input)
  }
)
