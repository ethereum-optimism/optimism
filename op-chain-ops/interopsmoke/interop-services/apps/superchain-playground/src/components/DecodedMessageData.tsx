import type { Abi, AbiParameter, Hex } from 'viem'
import { decodeFunctionData } from 'viem'

import { Label } from '@/components/ui/label'

const formatValue = (value: unknown, type: string): string | JSX.Element => {
  if (value === null || value === undefined) return 'null'

  // Handle tuples
  if (type.startsWith('tuple')) {
    const components = value as Record<string, unknown>
    return (
      <div className="pl-2 border-l-2 border-muted-foreground/20">
        {Object.entries(components).map(([key, val], idx) => (
          <div key={idx} className="flex gap-2">
            <span className="text-muted-foreground">{key}:</span>
            {formatValue(val, typeof val === 'object' ? 'tuple' : typeof val)}
          </div>
        ))}
      </div>
    )
  }

  // Handle arrays
  if (type.endsWith('[]')) {
    const baseType = type.slice(0, -2)
    const arrayValue = value as unknown[]
    return (
      <div className="pl-2 border-l-2 border-muted-foreground/20">
        {arrayValue.map((item, idx) => (
          <div key={idx} className="flex gap-2">
            <span className="text-muted-foreground">[{idx}]:</span>
            {formatValue(item, baseType)}
          </div>
        ))}
      </div>
    )
  }

  // Handle basic types
  switch (type) {
    case 'address':
      return value as string
    case 'bool':
      return value ? 'true' : 'false'
    case 'string':
      return `"${value}"`
    case 'bytes':
    case 'bytes32':
      return value as string
    default:
      // Handle uint/int types
      if (type.startsWith('uint') || type.startsWith('int')) {
        return value.toString()
      }
      return String(value)
  }
}

export const DecodedMessageData = ({
  message,
  abi,
}: {
  message: Hex
  abi: Abi
}) => {
  try {
    const { functionName, args } = decodeFunctionData({
      abi,
      data: message,
    })

    // Find the function in the ABI to get parameter names and types
    const functionAbi = abi.find(
      (item) => item.type === 'function' && item.name === functionName,
    )

    if (!functionAbi || !('inputs' in functionAbi)) {
      throw new Error('Function ABI not found')
    }

    const renderParameter = (
      input: AbiParameter,
      value: unknown,
      depth = 0,
    ) => {
      const formattedValue = formatValue(value, input.type)
      return (
        <div
          key={input.name}
          className="text-xs grid grid-cols-[auto_1fr] gap-2"
          style={{ marginLeft: `${depth * 8}px` }}
        >
          <div className="text-muted-foreground min-w-[150px]">
            <span>
              {input.name || '<unnamed>'}{' '}
              <span className="opacity-50">({input.type})</span>:
            </span>
          </div>
          <div className="break-all">{formattedValue}</div>
        </div>
      )
    }

    return (
      <div className="space-y-1 mt-2">
        <Label>Decoded Message Data</Label>
        <div className="rounded bg-muted p-2 space-y-2">
          <div className="text-xs font-medium">
            Function: <span className="text-primary">{functionName}</span>
          </div>
          <div className="space-y-1">
            {args &&
              functionAbi.inputs?.map((input, index) => {
                return renderParameter(input, args[index])
              })}
          </div>
        </div>
      </div>
    )
  } catch (error) {
    return (
      <div className="space-y-1 mt-2">
        <Label>Decoded Data</Label>
        <div className="rounded bg-muted p-2 text-xs text-destructive">
          Failed to decode message data: {(error as Error).message}
        </div>
      </div>
    )
  }
}
