import { useState } from 'react'
import { useScenarios, useRunScenario } from '@/hooks/useScenarios'
import { ScenarioCard } from '@/components/scenarios/ScenarioCard'
import { ScenarioDetail } from '@/components/scenarios/ScenarioDetail'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Search, Loader2 } from 'lucide-react'

export function Scenarios() {
  const { data, isLoading } = useScenarios()
  const runScenario = useRunScenario()
  const [search, setSearch] = useState('')
  const [selectedTag, setSelectedTag] = useState<string | null>(null)
  const [selectedScenario, setSelectedScenario] = useState<string | null>(null)

  // Extract unique tags
  const allTags = Array.from(
    new Set(data?.scenarios?.flatMap((s) => s.tags || []) || [])
  ).sort()

  // Filter scenarios
  const filteredScenarios = data?.scenarios?.filter((s) => {
    const matchesSearch = s.name.toLowerCase().includes(search.toLowerCase())
    const matchesTag = !selectedTag || s.tags?.includes(selectedTag)
    return matchesSearch && matchesTag
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Scenarios</h1>
        <span className="text-muted-foreground">{data?.count || 0} total</span>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search scenarios..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-10"
          aria-label="Search scenarios"
        />
      </div>

      {/* Tags */}
      {allTags.length > 0 && (
        <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by tag">
          <Badge
            variant={selectedTag === null ? 'default' : 'outline'}
            className="cursor-pointer"
            onClick={() => setSelectedTag(null)}
          >
            All
          </Badge>
          {allTags.map((tag) => (
            <Badge
              key={tag}
              variant={selectedTag === tag ? 'default' : 'outline'}
              className="cursor-pointer"
              onClick={() => setSelectedTag(tag === selectedTag ? null : tag)}
            >
              {tag}
            </Badge>
          ))}
        </div>
      )}

      {/* Scenario grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {filteredScenarios?.map((scenario) => (
          <ScenarioCard
            key={scenario.name}
            scenario={scenario}
            onRun={(name) => runScenario.mutate(name)}
            onSelect={setSelectedScenario}
          />
        ))}
      </div>

      {filteredScenarios?.length === 0 && (
        <div className="text-center text-muted-foreground">No scenarios found</div>
      )}

      {/* Detail modal */}
      {selectedScenario && (
        <ScenarioDetail name={selectedScenario} onClose={() => setSelectedScenario(null)} />
      )}
    </div>
  )
}
