import { useEffect } from 'react'
import { Routes, Route } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { Layout } from '@/components/layout/Layout'
import { Dashboard } from '@/pages/Dashboard'
import { Scenarios } from '@/pages/Scenarios'
import { Results } from '@/pages/Results'
import { Components } from '@/pages/Components'
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
 */
function SSEConnectionProvider({ children }: { children: React.ReactNode }) {
  useEventConnection()
  return <>{children}</>
}

export default function App() {
  return (
    <ModeDetector>
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
    </ModeDetector>
  )
}
