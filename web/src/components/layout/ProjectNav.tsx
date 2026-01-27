import {
  Folder,
  FolderOpen,
  Settings,
  Circle,
  CheckCircle2,
  XCircle,
  Loader2,
} from 'lucide-react'
import {
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from '@/components/ui/accordion'
import { NavItem } from './NavItem'
import { useProjectsStore } from '@/stores/projects'
import { useNavigationStore } from '@/stores/navigation'
import { cn } from '@/lib/utils'
import type { Project, ProjectConnectionStatus } from '@/api/types'

interface ProjectNavProps {
  project: Project
}

function StatusIcon({ status }: { status: ProjectConnectionStatus }) {
  switch (status) {
    case 'connected':
      return <CheckCircle2 className="h-3 w-3 text-green-500" />
    case 'connecting':
      return <Loader2 className="h-3 w-3 animate-spin text-yellow-500" />
    case 'error':
      return <XCircle className="h-3 w-3 text-red-500" />
    default:
      return <Circle className="h-3 w-3 text-muted-foreground" />
  }
}

export function ProjectNav({ project }: ProjectNavProps) {
  const expanded = useNavigationStore(state =>
    state.expandedProjects.has(project.id)
  )
  const toggleProject = useNavigationStore(state => state.toggleProject)
  const setActiveProject = useProjectsStore(state => state.setActiveProject)
  const activeProjectId = useProjectsStore(state => state.activeProjectId)

  const handleToggle = () => {
    toggleProject(project.id)
    setActiveProject(project.id)
  }

  const isActive = activeProjectId === project.id

  return (
    <AccordionItem>
      <AccordionTrigger
        expanded={expanded}
        onToggle={handleToggle}
        icon={expanded ? <FolderOpen className="h-4 w-4" /> : <Folder className="h-4 w-4" />}
        actions={<StatusIcon status={project.status} />}
        className={cn(isActive && 'bg-accent/50')}
      >
        {project.name}
      </AccordionTrigger>
      <AccordionContent expanded={expanded}>
        <SuitesList projectId={project.id} />
        <NavItem
          to={`/projects/${project.id}/settings`}
          icon={<Settings className="h-4 w-4" />}
          indent={1}
        >
          Settings
        </NavItem>
      </AccordionContent>
    </AccordionItem>
  )
}

interface SuitesListProps {
  projectId: string
}

function SuitesList({ projectId: _projectId }: SuitesListProps) {
  // This will be connected to React Query to fetch suites
  // For now, show placeholder
  return (
    <div className="py-1 text-xs text-muted-foreground pl-6">
      Loading suites...
    </div>
  )
}
