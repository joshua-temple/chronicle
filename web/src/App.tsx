import { Routes, Route } from 'react-router-dom'

export default function App() {
  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <Routes>
        <Route path="/" element={<div className="p-8 text-2xl">Chronicle Dashboard</div>} />
      </Routes>
    </div>
  )
}
