import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { Header } from './Header'
import { useMode } from '@/stores/mode'

export function Layout() {
  const mode = useMode()

  return (
    <div className="min-h-screen bg-background">
      <Sidebar mode={mode} />
      <div className="ml-64">
        <Header />
        <main className="p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
