import { DrippieVM, ReturnValue } from './vm'

export interface DrippieConfigV2 {
  [name: string]: {
    init: (vm: DrippieVM) => Array<ReturnValue | (() => Array<ReturnValue>)>
    check: (vm: DrippieVM) => ReturnValue
    actions: (vm: DrippieVM) => Array<ReturnValue | (() => Array<ReturnValue>)>
  }
}

export interface ParsedDrippieConfigV2 {
  [name: string]: {
    init: string[]
    checks: string[]
    actions: string[]
    stateI: string[]
    stateC: string[]
    stateA: string[]
  }
}

export const parseDrippieConfigV2 = (
  config: DrippieConfigV2
): ParsedDrippieConfigV2 => {
  const parsed: ParsedDrippieConfigV2 = {}
  for (const name of Object.keys(config)) {
    const { init, check, actions } = config[name]

    const initVM = new DrippieVM(name)
    const ret1 = init(initVM)
    for (const action of ret1) {
      if (typeof action === 'function') {
        action()
      }
    }
    const { commands: initCommands, state: initState } = initVM.compile()

    const checkVM = new DrippieVM(name)
    const ret2 = check(checkVM)
    checkVM.assert(ret2)
    const { commands: checkCommands, state: checkState } = checkVM.compile()

    const actionsVM = new DrippieVM(name)
    const ret3 = actions(actionsVM)
    for (const action of ret3) {
      if (typeof action === 'function') {
        action()
      }
    }
    const { commands: actionCommands, state: actionState } = actionsVM.compile()

    parsed[name] = {
      init: initCommands,
      checks: checkCommands,
      actions: actionCommands,
      stateI: initState,
      stateC: checkState,
      stateA: actionState,
    }
  }
  return parsed
}
