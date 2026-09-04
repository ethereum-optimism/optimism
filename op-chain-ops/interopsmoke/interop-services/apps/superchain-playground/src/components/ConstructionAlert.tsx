import { HardHat } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

export const ConstructionAlert = () => {
  return (
    <Alert>
      <HardHat className="h-4 w-4" />
      <AlertTitle>Under construction</AlertTitle>
      <AlertDescription>
        This page is still a work in progress.
      </AlertDescription>
    </Alert>
  )
}
