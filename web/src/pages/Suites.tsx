import { useState } from 'react'
import { useSuites, useRunSuite } from '@/hooks/useSuites'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { SkeletonList } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/ui/empty-state'
import { PlayCircle, FolderOpen, Tag, Hash } from 'lucide-react'
import type { Suite } from '@/api/types'

interface SuiteCardProps {
  suite: Suite
  onRun: (name: string) => void
  isRunning: boolean
}

function SuiteCard({ suite, onRun, isRunning }: SuiteCardProps) {
  const scenarioCount = suite.resolvedScenarios?.length || suite.scenarios?.length || 0

  return (
    <Card className="hover:border-primary/50 transition-colors">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center justify-between">
          <span className="flex items-center gap-2">
            <FolderOpen className="h-5 w-5 text-primary" />
            {suite.name}
          </span>
          <Button
            size="sm"
            onClick={() => onRun(suite.name)}
            disabled={isRunning}
          >
            <PlayCircle className="mr-2 h-4 w-4" />
            Run Suite
          </Button>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {suite.description && (
          <p className="text-sm text-muted-foreground">{suite.description}</p>
        )}

        <div className="flex items-center gap-4 text-sm text-muted-foreground">
          <span className="flex items-center gap-1">
            <Hash className="h-4 w-4" />
            {scenarioCount} scenario{scenarioCount !== 1 ? 's' : ''}
          </span>
          {suite.parallel && suite.parallel > 1 && (
            <span>Parallel: {suite.parallel}</span>
          )}
          {suite.failFast && (
            <Badge variant="outline" className="text-xs">Fail Fast</Badge>
          )}
        </div>

        {/* Tags */}
        {suite.tags && suite.tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            <Tag className="h-4 w-4 text-muted-foreground mr-1" />
            {suite.tags.map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}
          </div>
        )}

        {/* Exclude tags */}
        {suite.excludeTags && suite.excludeTags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            <span className="text-xs text-muted-foreground mr-1">Excludes:</span>
            {suite.excludeTags.map((tag: string) => (
              <Badge key={tag} variant="outline" className="text-xs text-red-500">
                {tag}
              </Badge>
            ))}
          </div>
        )}

        {/* Resolved scenarios (if available) */}
        {suite.resolvedScenarios && suite.resolvedScenarios.length > 0 && (
          <div className="mt-2 pt-2 border-t">
            <p className="text-xs text-muted-foreground mb-1">Scenarios:</p>
            <div className="flex flex-wrap gap-1">
              {suite.resolvedScenarios.slice(0, 5).map((name: string) => (
                <Badge key={name} variant="outline" className="text-xs">
                  {name}
                </Badge>
              ))}
              {suite.resolvedScenarios.length > 5 && (
                <Badge variant="outline" className="text-xs">
                  +{suite.resolvedScenarios.length - 5} more
                </Badge>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function Suites() {
  const { data, isLoading } = useSuites()
  const runSuite = useRunSuite()
  const [search, setSearch] = useState('')

  const filteredSuites = data?.suites?.filter((s) =>
    s.name.toLowerCase().includes(search.toLowerCase()) ||
    s.description?.toLowerCase().includes(search.toLowerCase())
  )

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">Suites</h1>
        </div>
        <SkeletonList count={4} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Suites</h1>
        <span className="text-muted-foreground">{data?.count || 0} total</span>
      </div>

      {/* Search */}
      {(data?.count ?? 0) > 0 && (
        <div className="relative">
          <input
            type="text"
            placeholder="Search suites..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            aria-label="Search suites"
          />
        </div>
      )}

      {/* Suite grid */}
      <div className="grid gap-4 md:grid-cols-2">
        {filteredSuites?.map((suite) => (
          <SuiteCard
            key={suite.name}
            suite={suite}
            onRun={(name) => runSuite.mutate(name)}
            isRunning={runSuite.isPending}
          />
        ))}
      </div>

      {filteredSuites?.length === 0 && (
        <EmptyState
          variant={search ? 'search' : 'empty'}
          title="No suites found"
          description={
            search
              ? 'Try adjusting your search criteria'
              : 'Define suites in your chronicle.yaml to group scenarios'
          }
          action={
            search
              ? {
                  label: 'Clear search',
                  onClick: () => setSearch(''),
                }
              : undefined
          }
        />
      )}
    </div>
  )
}
