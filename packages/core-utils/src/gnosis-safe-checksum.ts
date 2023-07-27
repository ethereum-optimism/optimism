import { ethers } from 'ethers'

// Slightly modified from:
// https://github.com/safe-global/safe-react-apps/blob/development/apps/tx-builder/src/lib/checksum.ts

const stringifyReplacer = (_: string, value: any) =>
  value === undefined ? null : value

const serializeJSONObject = (json: any): string => {
  if (Array.isArray(json)) {
    return `[${json.map((el) => serializeJSONObject(el)).join(',')}]`
  }

  if (typeof json === 'object' && json !== null) {
    let acc = ''
    const keys = Object.keys(json).sort()
    acc += `{${JSON.stringify(keys, stringifyReplacer)}`

    for (const key of keys) {
      acc += `${serializeJSONObject(json[key])},`
    }

    return `${acc}}`
  }

  return `${JSON.stringify(json, stringifyReplacer)}`
}

const calculateChecksum = (batchFile: any): string | undefined => {
  const serialized = serializeJSONObject({
    ...batchFile,
    meta: { ...batchFile.meta, name: null },
  })
  const sha = ethers.utils.solidityKeccak256(['string'], [serialized])

  return sha || undefined
}

export const addChecksum = (batchFile: any): any => {
  return {
    ...batchFile,
    meta: {
      ...batchFile.meta,
      checksum: calculateChecksum(batchFile),
    },
  }
}
