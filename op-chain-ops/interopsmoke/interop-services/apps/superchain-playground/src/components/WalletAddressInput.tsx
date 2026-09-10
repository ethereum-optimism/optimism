import { X } from 'lucide-react'
import { useState } from 'react'
import type { Address } from 'viem'
import { isAddress } from 'viem'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export const WalletAddressInput = ({
  addresses,
  setAddresses,
}: {
  addresses: Address[]
  setAddresses: (addresses: Address[]) => void
}) => {
  const [inputValue, setInputValue] = useState('')
  const [error, setError] = useState<string | null>(null)

  const addAddress = (addressToAdd: string) => {
    if (isAddress(addressToAdd)) {
      const normalizedAddress = addressToAdd as Address
      if (addresses.includes(normalizedAddress)) {
        setError(`Address ${addressToAdd} already added`)
        return false
      }
      setAddresses([...addresses, normalizedAddress])
      return true
    } else {
      setError(`Invalid Ethereum address: ${addressToAdd}`)
      return false
    }
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value)
    setError(null)
  }

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && inputValue) {
      e.preventDefault()
      const values = inputValue
        .split(',')
        .map((v) => v.trim())
        .filter(Boolean)

      if (values.length === 0) return

      let allValid = true
      for (const value of values) {
        const success = addAddress(value)
        if (!success) {
          allValid = false
          break
        }
      }

      if (allValid) {
        setInputValue('')
        setError(null)
      }
    }
  }

  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault()
    const pastedText = e.clipboardData.getData('text')
    const values = pastedText
      .split(/[,\n]/)
      .map((v) => v.trim())
      .filter(Boolean)

    if (values.length === 0) return

    const newAddresses = [...addresses]
    const invalidAddresses: string[] = []
    let anyAdded = false

    for (const value of values) {
      if (isAddress(value)) {
        const normalizedAddress = value as Address
        if (!addresses.includes(normalizedAddress)) {
          newAddresses.push(normalizedAddress)
          anyAdded = true
        }
      } else {
        invalidAddresses.push(value)
      }
    }

    // Only show error for invalid addresses
    if (invalidAddresses.length > 0) {
      setError(`Invalid addresses: ${invalidAddresses.join(', ')}`)
    } else {
      setError(null)
    }

    // Update addresses if we added any new ones
    if (anyAdded) {
      setAddresses(newAddresses)
      setInputValue('')
    }
  }

  const removeAddress = (addressToRemove: Address) => {
    setAddresses(addresses.filter((addr) => addr !== addressToRemove))
  }

  return (
    <div className="flex flex-col gap-2">
      <Label>Recipient Addresses</Label>
      <Input
        placeholder="Enter Ethereum addresses (comma separated)"
        value={inputValue}
        onChange={handleInputChange}
        onKeyDown={handleInputKeyDown}
        onPaste={handlePaste}
        className="text-sm"
      />
      {error && <p className="text-sm text-destructive">{error}</p>}
      <div className="flex flex-wrap gap-2 mt-2">
        {addresses.map((address) => (
          <div
            key={address}
            className="flex items-center gap-2 bg-muted/30 px-3 py-1.5 rounded-lg text-sm"
          >
            <span className="font-mono">
              {`${address.slice(0, 6)}...${address.slice(-4)}`}
            </span>
            <button
              type="button"
              onClick={() => removeAddress(address)}
              className="text-muted-foreground hover:text-destructive transition-colors"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
