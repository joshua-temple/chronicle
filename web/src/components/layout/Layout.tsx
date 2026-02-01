import { Outlet } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { Sidebar } from './Sidebar'
import { Header } from './Header'
import { ToastContainer } from '@/components/ui/toast'
import { Button } from '@/components/ui/button'
import { useActiveProject } from '@/stores/projects'

interface LayoutProps {
  onBackToProjects?: () => void
}

export function Layout({ onBackToProjects }: LayoutProps) {
  const activeProject = useActiveProject()

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <div className="ml-64">
        {onBackToProjects && (
          <div className="border-b border-border bg-muted/50 px-6 py-2">
            <div className="flex items-center gap-4">
              <Button
                variant="ghost"
                size="sm"
                onClick={onBackToProjects}
                className="gap-2"
              >
                <ArrowLeft className="h-4 w-4" />
                Back to Projects
              </Button>
              {activeProject && (
                <span className="text-sm text-muted-foreground">
                  Working on: <span className="font-medium text-foreground">{activeProject.name}</span>
                </span>
              )}
            </div>
          </div>
        )}
        <Header />
        <main className="p-6">
          <Outlet />
        </main>
      </div>
      <ToastContainer />
    </div>
  )
}
