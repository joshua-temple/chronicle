import { useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { Layout } from '@/components/layout/Layout'
import { Dashboard } from '@/pages/Dashboard'
import { Scenarios } from '@/pages/Scenarios'
import { Suites } from '@/pages/Suites'
import { Results } from '@/pages/Results'
import { Components } from '@/pages/Components'
import { ConfigEditor } from '@/pages/ConfigEditor'
import { ProjectSelector } from '@/components/standalone'
import { useEventConnection } from '@/hooks/useEvents'
import { useModeStore, useMode } from '@/stores/mode'
import { useProjectsStore } from '@/stores/projects'

function Runs() {
  return <div className="text-2xl font-bold">Runs</div>
}

/**
 * Mode Detector component
 * Detects whether the app is running in standalone or daemon mode
 *
 * IMPORTANT: The UI always loads regardless of daemon detection.
 * The UI is designed to be a controller for multiple daemons/projects.
 * When no daemon is detected, it defaults to standalone mode for project management.
 */
function ModeDetector({ children }: { children: React.ReactNode }) {
  const mode = useMode()
  const detectMode = useModeStore((state) => state.detectMode)

  useEffect(() => {
    detectMode()
  }, [detectMode])

  // Show brief loading state only during initial detection
  if (mode === 'detecting') {
    return (
      <div className="flex h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Always render children - UI should always be accessible
  // The disconnected state is handled within the app by showing
  // project/daemon management UI instead of blocking
  return <>{children}</>
}

/**
 * SSE Connection Provider component
 * Establishes SSE connection at app level and maintains it throughout the session
 * Only used in daemon mode since standalone mode doesn't have SSE
 */
function SSEConnectionProvider({ children }: { children: React.ReactNode }) {
  useEventConnection()
  return <>{children}</>
}

/**
 * Standalone mode routes
 * Config-focused routes for local project editing without daemon
 * Shows ProjectSelector when no active project is selected
 */
function StandaloneRoutes() {
  const activeProjectId = useProjectsStore((state) => state.activeProjectId)
  const setActiveProject = useProjectsStore((state) => state.setActiveProject)

  // If no active project, show project selector (not wrapped in Layout)
  if (!activeProjectId) {
    return <ProjectSelector />
  }

  // Active project selected - show project UI with back navigation
  return (
    <Routes>
      <Route element={<Layout onBackToProjects={() => setActiveProject(null)} />}>
        <Route path="/" element={<Navigate to="/config" replace />} />
        <Route path="/config" element={<ConfigEditor />} />
        <Route path="/scenarios" element={<Scenarios />} />
        <Route path="/components" element={<Components />} />
        {/* Redirect daemon-only routes to config */}
        <Route path="/runs" element={<Navigate to="/config" replace />} />
        <Route path="/results" element={<Navigate to="/config" replace />} />
      </Route>
    </Routes>
  )
}

/**
 * Daemon mode routes
 * Full functionality with runs, results, and SSE events
 */
function DaemonRoutes() {
  return (
    <SSEConnectionProvider>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/scenarios" element={<Scenarios />} />
          <Route path="/suites" element={<Suites />} />
          <Route path="/runs" element={<Runs />} />
          <Route path="/results" element={<Results />} />
          <Route path="/components" element={<Components />} />
        </Route>
      </Routes>
    </SSEConnectionProvider>
  )
}

/**
 * App Router
 * Renders routes based on detected mode
 *
 * When disconnected, defaults to standalone mode for project management.
 * This allows users to configure projects even without a running daemon.
 */
function AppRouter() {
  const mode = useMode()

  // Standalone mode or disconnected - show project management UI
  if (mode === 'standalone' || mode === 'disconnected') {
    return <StandaloneRoutes />
  }

  // Daemon mode - full functionality
  return <DaemonRoutes />
}

export default function App() {
  return (
    <ModeDetector>
      <AppRouter />
    </ModeDetector>
  )
}
