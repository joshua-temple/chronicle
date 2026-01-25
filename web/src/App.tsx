import { Routes, Route } from 'react-router-dom'
import { Layout } from '@/components/layout/Layout'
import { Dashboard } from '@/pages/Dashboard'
import { Scenarios } from '@/pages/Scenarios'
import { Results } from '@/pages/Results'
import { Components } from '@/pages/Components'
import { useEventConnection } from '@/hooks/useEvents'

function Runs() {
  return <div className="text-2xl font-bold">Runs</div>
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
