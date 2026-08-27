import { ConnectWalletButton } from '@/components/ConnectWalletButton'
import { SidebarTrigger, useSidebar } from '@/components/ui/sidebar'

export const NavBar = () => {
  const { isMobile } = useSidebar()
  return (
    <div className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 h-16 flex justify-between items-center px-4">
      {isMobile ? <SidebarTrigger className="" /> : <div />}
      <div className="flex items-center gap-4 select-none">
        <ConnectWalletButton />
      </div>
    </div>
  )
}
