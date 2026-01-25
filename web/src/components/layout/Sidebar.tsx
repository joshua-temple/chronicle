import { NavLink } from 'react-router-dom'
import { LayoutDashboard, PlayCircle, History, FileText, Boxes, Settings } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AppMode } from '@/stores/mode'

interface SidebarProps {
  mode: AppMode
}

const STANDALONE_NAV = [
  { to: '/config', icon: Settings, label: 'Config' },
  { to: '/scenarios', icon: FileText, label: 'Scenarios' },
  { to: '/components', icon: Boxes, label: 'Components' },
]

const DAEMON_NAV = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/scenarios', icon: PlayCircle, label: 'Scenarios' },
  { to: '/runs', icon: History, label: 'Runs' },
  { to: '/results', icon: FileText, label: 'Results' },
  { to: '/components', icon: Boxes, label: 'Components' },
]

export function Sidebar({ mode }: SidebarProps) {
  const navItems = mode === 'standalone' ? STANDALONE_NAV : DAEMON_NAV
  return (
    <aside className="fixed left-0 top-0 z-40 h-screen w-64 border-r border-border bg-card">
      <div className="flex h-16 items-center border-b border-border px-6">
        <span className="text-xl font-bold text-primary">Chronicle</span>
      </div>
      <nav aria-label="Main navigation" className="space-y-1 p-4">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-secondary text-secondary-foreground'
                  : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground'
              )
            }
          >
            <item.icon className="h-5 w-5" />
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
