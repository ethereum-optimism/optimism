import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'

import { NavBarLayout } from '@/layouts/NavBarLayout'
import { BridgePage } from '@/pages/BridgePage'
import { ChainsPage } from '@/pages/ChainsPage'
import { ConfigPage } from '@/pages/ConfigPage'
import { L2ToL1RelayerPage } from '@/pages/L2ToL1RelayerPage'
import { SuperchainETHBridgePage } from '@/pages/SuperchainETHBridgePage'
import { SuperchainMessageRelayer } from '@/pages/SuperchainMessageRelayer'
import { SuperchainTokenBridgePage } from '@/pages/SuperchainTokenBridgePage'

const router = createBrowserRouter([
  {
    element: <NavBarLayout />,
    children: [
      {
        path: '/',
        element: <Navigate to="/bridge" replace />,
      },
      {
        path: '/bridge',
        element: <BridgePage />,
      },
      {
        path: '/chains',
        element: <ChainsPage />,
      },
      {
        path: '/l2-to-l1-relayer',
        element: <L2ToL1RelayerPage />,
      },
      {
        path: '/superchain-eth-bridge',
        element: <SuperchainETHBridgePage />,
      },
      {
        path: '/superchain-token-bridge',
        element: <SuperchainTokenBridgePage />,
      },
      {
        path: '/superchain-message-relayer',
        element: <SuperchainMessageRelayer />,
      },
      {
        path: '/config',
        element: <ConfigPage />,
      },
    ],
  },
])

export const AppRoutes = () => {
  return <RouterProvider router={router} />
}
