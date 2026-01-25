import { Routes, Route } from 'react-router-dom'
import { Layout } from '@/components/layout/Layout'
import { Dashboard } from '@/pages/Dashboard'

function Scenarios() {
  return <div className="text-2xl font-bold">Scenarios</div>
}

function Runs() {
  return <div className="text-2xl font-bold">Runs</div>
}

function Results() {
  return <div className="text-2xl font-bold">Results</div>
}

function Components() {
  return <div className="text-2xl font-bold">Components</div>
}

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/scenarios" element={<Scenarios />} />
        <Route path="/runs" element={<Runs />} />
        <Route path="/results" element={<Results />} />
        <Route path="/components" element={<Components />} />
      </Route>
    </Routes>
  )
}
