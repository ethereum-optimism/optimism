import { parseDrippieConfigV2 } from './src'
import * as config from './config/drippie-v2/optimism-kovan'

const main = async () => {
  const parsed = parseDrippieConfigV2((config as any).default)
  console.log(parsed)

  const fmt = (a: any[]) => {
    return `[${a
      .map((x) => {
        return `"${x.toString()}"`
      })
      .join(',')}]`
  }

  console.log(
    `[${fmt(parsed.SimpleBalance.init)},${fmt(
      parsed.SimpleBalance.checks
    )},${fmt(parsed.SimpleBalance.actions)},${fmt(
      parsed.SimpleBalance.stateI
    )},${fmt(parsed.SimpleBalance.stateC)},${fmt(parsed.SimpleBalance.stateA)}]`
  )

  // console.log(
  //   `[1, [${commands.map((c) => `"${c.toString()}"`).join(', ')}],[${commands2
  //     .map((c) => `"${c.toString()}"`)
  //     .join(', ')}],[${state
  //     .map((s) => `"${s.toString()}"`)
  //     .join(', ')}],[${state2.map((s) => `"${s.toString()}"`).join(', ')}]]`
  // )
}

main()
