import { Outlet } from 'react-router-dom'

import { NavBar } from '@/components/NavBar'
import { SideNav } from '@/components/SideNav'
import { SidebarProvider } from '@/components/ui/sidebar'

export const NavBarLayout = () => {
  return (
    <SidebarProvider className="flex-1 flex flex-col">
      <div className="flex flex-col">
        <div className="flex flex-1">
          <SideNav />
          <main className="flex flex-col flex-1">
            <NavBar />
            <div className="flex flex-1 flex-col p-8">
              <Outlet />
            </div>
          </main>
        </div>
      </div>
    </SidebarProvider>
  )
}
