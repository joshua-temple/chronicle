import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'

async function fetchHealth() {
  const res = await fetch('/api/v1/health')
  if (!res.ok) throw new Error('Health check failed')
  return res.json()
}

export function Header() {
  const { data: health, isLoading, refetch } = useQuery({
    queryKey: ['health'],
    queryFn: fetchHealth,
    refetchInterval: 30000,
  })

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-card/95 px-6 backdrop-blur">
      <div className="flex items-center gap-4">
        <h1 className="text-lg font-semibold">Test Orchestration</h1>
      </div>
      <div className="flex items-center gap-4">
        <Badge variant={health?.status === 'healthy' ? 'success' : 'destructive'}>
          {isLoading ? 'Checking...' : health?.status || 'Unknown'}
        </Badge>
        <Button variant="ghost" size="icon" onClick={() => refetch()}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>
    </header>
  )
}
