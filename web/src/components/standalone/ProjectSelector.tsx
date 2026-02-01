import { useEffect, useState, useCallback } from 'react'
import { Plus, RefreshCw, Search } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { ProjectCard } from './ProjectCard'
import { AddProjectModal } from './AddProjectModal'
import {
  useProjectsStore,
  useProjects,
} from '@/stores/projects'
import type { Project } from '@/api/types'

export function ProjectSelector() {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [isDiscovering, setIsDiscovering] = useState(false)

  const projects = useProjects()
  const discovered = useProjectsStore(state => state.discovered)
  const loading = useProjectsStore(state => state.loading)
  const discovering = useProjectsStore(state => state.discovering)
  const error = useProjectsStore(state => state.error)

  const {
    addProject,
    removeProject,
    connectProject,
    disconnectProject,
    setActiveProject,
    runDiscovery,
    addDiscovered,
    clearError,
    loadProjects,
    connectAll,
  } = useProjectsStore()

  // Load projects on mount
  useEffect(() => {
    loadProjects()
  }, [loadProjects])

  // Auto-connect to all projects on mount
  useEffect(() => {
    if (projects.length > 0) {
      connectAll()
    }
  }, []) // Only run once on mount

  const handleRefresh = useCallback(async () => {
    await connectAll()
  }, [connectAll])

  const handleDiscover = useCallback(async () => {
    setIsDiscovering(true)
    try {
      await runDiscovery()
    } finally {
      setIsDiscovering(false)
    }
  }, [runDiscovery])

  const handleAddProject = useCallback(
    async (projectData: { name: string; path?: string; remoteUrl?: string }) => {
      // Convert modal output to the store's expected format
      // For remote projects, remoteUrl is the daemon URL
      // For local projects, the path could be used to construct a localhost URL or stored for reference
      const daemonUrl = projectData.remoteUrl || `http://localhost:8080`

      // addProject will throw if it fails, allowing the modal to handle the error
      addProject({
        name: projectData.name,
        daemonUrl: daemonUrl,
        description: projectData.path ? `Project path: ${projectData.path}` : undefined,
      })
      // Only close on success
      setIsAddModalOpen(false)
    },
    [addProject]
  )

  const handleOpen = useCallback(
    (id: string) => {
      setActiveProject(id)
    },
    [setActiveProject]
  )

  const handleAddDiscoveredProject = useCallback(
    (project: Project) => {
      addDiscovered(project)
    },
    [addDiscovered]
  )

  // Filter out discovered projects that are already registered
  const registeredUrls = new Set(projects.map((p) => p.daemonUrl))
  const filteredDiscovered = discovered.filter((p) => !registeredUrls.has(p.daemonUrl))

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border bg-card">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-foreground">Chronicle Control Center</h1>
              <p className="text-sm text-muted-foreground mt-1">
                Manage your Chronicle projects
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleRefresh}
                disabled={loading}
                aria-label="Refresh projects"
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Button onClick={() => setIsAddModalOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Add Project
              </Button>
            </div>
          </div>
        </div>
      </header>

      <main className="container mx-auto px-4 py-6">
        {/* Error Display */}
        {error && (
          <div className="mb-6 rounded-lg border border-destructive/50 bg-destructive/10 p-4">
            <div className="flex items-center justify-between">
              <p className="text-sm text-destructive">{error}</p>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={clearError}>
                  Dismiss
                </Button>
                <Button variant="outline" size="sm" onClick={handleRefresh}>
                  Retry
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Loading State */}
        {loading && projects.length === 0 && (
          <div className="space-y-4">
            <ProjectCardSkeleton />
            <ProjectCardSkeleton />
            <ProjectCardSkeleton />
          </div>
        )}

        {/* Registered Projects */}
        {!loading && projects.length === 0 && !error ? (
          <EmptyState
            variant="empty"
            title="No projects yet"
            description="Add a Chronicle project to get started. You can also search for existing Chronicle daemons on your network."
            action={{
              label: 'Add Project',
              onClick: () => setIsAddModalOpen(true),
            }}
          />
        ) : projects.length > 0 ? (
          <section className="mb-8">
            <h2 className="text-lg font-semibold mb-4">Your Projects</h2>
            <div className="space-y-4">
              {projects.map((project) => (
                <ProjectCard
                  key={project.id}
                  project={project}
                  onOpen={handleOpen}
                  onConnect={connectProject}
                  onDisconnect={disconnectProject}
                  onRemove={removeProject}
                  disabled={loading}
                />
              ))}
            </div>
          </section>
        ) : null}

        {/* Discovered Projects */}
        {filteredDiscovered.length > 0 && (
          <section>
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-semibold">Discovered Projects</h2>
                <p className="text-sm text-muted-foreground">
                  Chronicle daemons found on your network
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleDiscover}
                disabled={isDiscovering || discovering || loading}
              >
                <Search className={`h-4 w-4 mr-2 ${isDiscovering || discovering ? 'animate-pulse' : ''}`} />
                Scan Again
              </Button>
            </div>
            <div className="space-y-4">
              {filteredDiscovered.map((project) => (
                <DiscoveredProjectCard
                  key={project.id}
                  project={project}
                  onAdd={handleAddDiscoveredProject}
                  disabled={loading}
                />
              ))}
            </div>
          </section>
        )}

        {/* Discover Hint */}
        {projects.length > 0 && filteredDiscovered.length === 0 && (
          <div className="mt-8 text-center">
            <Button
              variant="ghost"
              onClick={handleDiscover}
              disabled={isDiscovering || discovering || loading}
            >
              <Search className={`h-4 w-4 mr-2 ${isDiscovering || discovering ? 'animate-pulse' : ''}`} />
              Scan for Chronicle daemons
            </Button>
          </div>
        )}
      </main>

      {/* Add Project Modal */}
      <AddProjectModal
        open={isAddModalOpen}
        onClose={() => setIsAddModalOpen(false)}
        onSubmit={handleAddProject}
        loading={loading}
      />
    </div>
  )
}

// Skeleton loader for project cards
function ProjectCardSkeleton() {
  return (
    <div className="rounded-lg border border-border bg-card p-4 shadow-sm">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 flex-1">
          <Skeleton className="mt-1.5 h-3 w-3 rounded-full" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-4 w-64" />
          </div>
        </div>
        <div className="space-y-2 text-right">
          <Skeleton className="h-4 w-24 ml-auto" />
          <Skeleton className="h-3 w-20 ml-auto" />
        </div>
      </div>
      <div className="mt-4 flex justify-end gap-2">
        <Skeleton className="h-8 w-8" />
        <Skeleton className="h-8 w-20" />
        <Skeleton className="h-8 w-16" />
      </div>
    </div>
  )
}

// Compact card for discovered projects
interface DiscoveredProjectCardProps {
  project: Project
  onAdd: (project: Project) => void
  disabled?: boolean
}

function DiscoveredProjectCard({ project, onAdd, disabled }: DiscoveredProjectCardProps) {
  return (
    <div className="rounded-lg border border-dashed border-border bg-card/50 p-4">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h3 className="font-medium text-foreground">{project.name}</h3>
          <p className="text-sm text-muted-foreground truncate">{project.daemonUrl}</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onAdd(project)}
          disabled={disabled}
        >
          <Plus className="h-4 w-4 mr-1" />
          Add
        </Button>
      </div>
    </div>
  )
}
