import { AppRoutes } from '@/AppRouter'
import { Toaster } from '@/components/ui/toaster'
import { Providers } from '@/Providers'

function App() {
  return (
    <Providers>
      <Toaster />
      <AppRoutes />
    </Providers>
  )
}

export default App
