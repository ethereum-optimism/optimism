import { cva, type VariantProps } from 'class-variance-authority'
import * as React from 'react'

import { cn } from '@/lib/utils'

export const chipVariants = cva(
  'inline-flex select-none rounded-full font-semibold ',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        success: 'bg-blue-600 text-white hover:bg-600/90',
        progress: 'bg-emerald-600 text-white hover:bg-emerald-600/90',
        error:
          'bg-destructive text-destructive-foreground hover:bg-destructive/9',
      },
      size: {
        default: 'py-1 px-4 text-md',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

export interface ChipProps
  extends React.ButtonHTMLAttributes<HTMLDivElement>,
    VariantProps<typeof chipVariants> {}

export const Chip = React.forwardRef<HTMLDivElement, ChipProps>(
  ({ className, variant, size, ...props }, ref) => {
    return (
      <div
        className={cn(chipVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  },
)
