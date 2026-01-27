import {
  Layers,
  Puzzle,
  Play,
  FileText,
  Settings,
  CheckSquare,
} from 'lucide-react'
import {
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from '@/components/ui/accordion'
import { NavItem } from './NavItem'
import { useNavigationStore } from '@/stores/navigation'
import type { Suite } from '@/api/types'

interface SuiteNavProps {
  suite: Suite
  projectId: string
}

export function SuiteNav({ suite, projectId }: SuiteNavProps) {
  const suiteId = `${projectId}:${suite.name}`
  const expanded = useNavigationStore(state =>
    state.expandedSuites.has(suiteId)
  )
  const toggleSuite = useNavigationStore(state => state.toggleSuite)

  const handleToggle = () => {
    toggleSuite(suiteId)
  }

  const basePath = `/projects/${projectId}/suites/${encodeURIComponent(suite.name)}`

  return (
    <AccordionItem>
      <AccordionTrigger
        expanded={expanded}
        onToggle={handleToggle}
        icon={<Layers className="h-4 w-4" />}
        actions={
          <span className="text-xs text-muted-foreground">
            {suite.scenarioCount || suite.scenarios?.length || 0}
          </span>
        }
      >
        {suite.name}
      </AccordionTrigger>
      <AccordionContent expanded={expanded}>
        <NavItem
          to={`${basePath}/scenarios`}
          icon={<CheckSquare className="h-4 w-4" />}
          indent={2}
        >
          Scenarios
        </NavItem>
        <NavItem
          to={`${basePath}/plugins`}
          icon={<Puzzle className="h-4 w-4" />}
          indent={2}
        >
          Plugins
        </NavItem>
        <NavItem
          to={`${basePath}/runs`}
          icon={<Play className="h-4 w-4" />}
          indent={2}
        >
          Runs
        </NavItem>
        <NavItem
          to={`${basePath}/results`}
          icon={<FileText className="h-4 w-4" />}
          indent={2}
        >
          Results
        </NavItem>
        <NavItem
          to={`${basePath}/config`}
          icon={<Settings className="h-4 w-4" />}
          indent={2}
        >
          Config
        </NavItem>
      </AccordionContent>
    </AccordionItem>
  )
}
