import { useEffect } from 'react'
import {
  LayoutDashboard,
  Settings,
  Plus,
  RefreshCw,
  Loader2,
} from 'lucide-react'
import { Accordion } from '@/components/ui/accordion'
import { NavItem, NavGroup, NavDivider } from './NavItem'
import { ProjectNav } from './ProjectNav'
import { Button } from '@/components/ui/button'
import { useProjectsStore } from '@/stores/projects'
import { useSidebarCollapsed } from '@/stores/settings'

export function Sidebar() {
  const collapsed = useSidebarCollapsed()
  const projects = useProjectsStore(state => state.projects)
  const discovered = useProjectsStore(state => state.discovered)
  const loading = useProjectsStore(state => state.loading)
  const discovering = useProjectsStore(state => state.discovering)
  const loadProjects = useProjectsStore(state => state.loadProjects)
  const runDiscovery = useProjectsStore(state => state.runDiscovery)
  const connectAll = useProjectsStore(state => state.connectAll)

  // Load projects on mount
  useEffect(() => {
    loadProjects()
  }, [loadProjects])

  // Auto-connect on load
  useEffect(() => {
    if (projects.length > 0) {
      connectAll()
    }
  }, [projects.length, connectAll])

  if (collapsed) {
    return (
      <aside className="w-12 border-r border-border bg-background">
        {/* Collapsed sidebar - just icons */}
      </aside>
    )
  }

  return (
    <aside className="w-64 border-r border-border bg-background flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-border">
        <h1 className="text-lg font-bold">Chronicle</h1>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto p-2 space-y-4">
        {/* Global Navigation */}
        <NavGroup label="Overview">
          <NavItem
            to="/"
            icon={<LayoutDashboard className="h-4 w-4" />}
          >
            Dashboard
          </NavItem>
        </NavGroup>

        <NavDivider />

        {/* Projects */}
        <NavGroup label="Projects">
          {loading ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="h-4 w-4 animate-spin" />
            </div>
          ) : projects.length === 0 ? (
            <div className="px-2 py-4 text-sm text-muted-foreground text-center">
              No projects yet
            </div>
          ) : (
            <Accordion>
              {projects.map(project => (
                <ProjectNav key={project.id} project={project} />
              ))}
            </Accordion>
          )}

          {/* Discovered projects */}
          {discovered.length > 0 && (
            <div className="mt-2 space-y-1">
              <div className="px-2 text-xs text-muted-foreground">
                Discovered ({discovered.length})
              </div>
              {discovered.map(project => (
                <DiscoveredProjectItem
                  key={project.id}
                  project={project}
                />
              ))}
            </div>
          )}
        </NavGroup>

        <NavDivider />

        {/* Settings */}
        <NavGroup label="Settings">
          <NavItem
            to="/settings"
            icon={<Settings className="h-4 w-4" />}
          >
            UI Settings
          </NavItem>
        </NavGroup>
      </nav>

      {/* Footer Actions */}
      <div className="p-2 border-t border-border space-y-1">
        <Button
          variant="ghost"
          size="sm"
          className="w-full justify-start"
          onClick={() => runDiscovery()}
          disabled={discovering}
        >
          {discovering ? (
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4 mr-2" />
          )}
          Discover
        </Button>
        <a
          href="/projects/add"
          className="flex items-center w-full justify-start text-sm h-9 rounded-md px-3 hover:bg-accent hover:text-accent-foreground"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Project
        </a>
      </div>
    </aside>
  )
}

interface DiscoveredProjectItemProps {
  project: { id: string; name: string; daemonUrl: string }
}

function DiscoveredProjectItem({ project }: DiscoveredProjectItemProps) {
  const addDiscovered = useProjectsStore(state => state.addDiscovered)
  const dismissDiscovered = useProjectsStore(state => state.dismissDiscovered)

  return (
    <div className="flex items-center gap-2 px-2 py-1 text-sm">
      <span className="flex-1 truncate text-muted-foreground">
        {project.name}
      </span>
      <Button
        variant="ghost"
        size="sm"
        className="h-6 px-2"
        onClick={() => addDiscovered(project as any)}
      >
        Add
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className="h-6 px-2"
        onClick={() => dismissDiscovered(project.id)}
      >
        Dismiss
      </Button>
    </div>
  )
}
