import { useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { Layout } from '@/components/layout/Layout'
import { Dashboard } from '@/pages/Dashboard'
import { Scenarios } from '@/pages/Scenarios'
import { Results } from '@/pages/Results'
import { Components } from '@/pages/Components'
import { ConfigEditor } from '@/pages/ConfigEditor'
import { useEventConnection } from '@/hooks/useEvents'
import { useModeStore, useMode } from '@/stores/mode'

function Runs() {
  return <div className="text-2xl font-bold">Runs</div>
}

/**
 * Mode Detector component
 * Detects whether the app is running in standalone or daemon mode
 */
function ModeDetector({ children }: { children: React.ReactNode }) {
  const mode = useMode()
  const detectMode = useModeStore((state) => state.detectMode)

  useEffect(() => {
    detectMode()
  }, [detectMode])

  if (mode === 'detecting') {
    return (
      <div className="flex h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (mode === 'disconnected') {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-center">
          <h1 className="text-xl font-semibold">Not Connected</h1>
          <p className="text-muted-foreground mt-2">
            Start Chronicle with <code className="bg-muted px-1 rounded">chronicle ui</code> or <code className="bg-muted px-1 rounded">chronicle daemon</code>
          </p>
        </div>
      </div>
    )
  }

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
 */
function StandaloneRoutes() {
  return (
    <Routes>
      <Route element={<Layout />}>
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
 */
function AppRouter() {
  const mode = useMode()

  if (mode === 'standalone') {
    return <StandaloneRoutes />
  }

  return <DaemonRoutes />
}

export default function App() {
  return (
    <ModeDetector>
      <AppRouter />
    </ModeDetector>
  )
}
