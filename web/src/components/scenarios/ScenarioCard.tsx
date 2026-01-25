import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Highlight } from '@/components/ui/highlight'
import { PlayCircle, ChevronRight } from 'lucide-react'
import type { Scenario } from '@/api/types'

interface ScenarioCardProps {
  scenario: Scenario
  onRun: (name: string) => void
  onSelect: (name: string) => void
  searchQuery?: string
}

export function ScenarioCard({ scenario, onRun, onSelect, searchQuery = '' }: ScenarioCardProps) {
  return (
    <Card
      className="cursor-pointer transition-colors hover:bg-secondary/50"
      onClick={() => onSelect(scenario.name)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onSelect(scenario.name)}
    >
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">
            <Highlight text={scenario.name} query={searchQuery} />
          </CardTitle>
          <Button
            variant="ghost"
            size="icon"
            aria-label={`Run ${scenario.name}`}
            onClick={(e) => {
              e.stopPropagation()
              onRun(scenario.name)
            }}
          >
            <PlayCircle className="h-5 w-5" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {scenario.description && (
          <p className="mb-2 text-sm text-muted-foreground">
            <Highlight text={scenario.description} query={searchQuery} />
          </p>
        )}
        <div className="flex items-center justify-between">
          <div className="flex flex-wrap gap-1">
            {scenario.tags?.slice(0, 3).map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}
            {(scenario.tags?.length || 0) > 3 && (
              <Badge variant="outline" className="text-xs">
                +{scenario.tags!.length - 3}
              </Badge>
            )}
          </div>
          <div className="flex items-center text-sm text-muted-foreground">
            {scenario.flow_count} {scenario.flow_count === 1 ? 'step' : 'steps'}
            <ChevronRight className="ml-1 h-4 w-4" />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
