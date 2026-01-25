import { useState } from 'react'
import { useComponents, useComponent } from '@/hooks/useComponents'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Search, Loader2, X, FileCode, ChevronRight } from 'lucide-react'
import type { Component } from '@/api/types'

type ComponentType = 'all' | 'setup' | 'task' | 'validation' | 'teardown'

const TYPE_LABELS: Record<ComponentType, string> = {
  all: 'All',
  setup: 'Setup',
  task: 'Task',
  validation: 'Validation',
  teardown: 'Teardown',
}

const TYPE_COLORS: Record<Component['type'], string> = {
  setup: 'bg-blue-500/20 text-blue-400',
  task: 'bg-purple-500/20 text-purple-400',
  validation: 'bg-amber-500/20 text-amber-400',
  teardown: 'bg-rose-500/20 text-rose-400',
}

export function Components() {
  const { data, isLoading } = useComponents()
  const [search, setSearch] = useState('')
  const [selectedType, setSelectedType] = useState<ComponentType>('all')
  const [selectedComponent, setSelectedComponent] = useState<string | null>(null)

  // Filter components
  const filteredComponents = data?.components?.filter((c) => {
    const matchesSearch = c.name.toLowerCase().includes(search.toLowerCase())
    const matchesType = selectedType === 'all' || c.type === selectedType
    return matchesSearch && matchesType
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-8 w-8 animate-spin" aria-label="Loading components" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Components</h1>
        <span className="text-muted-foreground">{data?.count || 0} total</span>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search components..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-10"
          aria-label="Search components"
        />
      </div>

      {/* Type filter buttons */}
      <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by type">
        {(Object.keys(TYPE_LABELS) as ComponentType[]).map((type) => (
          <Button
            key={type}
            variant={selectedType === type ? 'default' : 'outline'}
            size="sm"
            onClick={() => setSelectedType(type)}
            aria-pressed={selectedType === type}
            aria-label={`Filter by ${TYPE_LABELS[type]}`}
          >
            {TYPE_LABELS[type]}
          </Button>
        ))}
      </div>

      {/* Component grid */}
      {filteredComponents?.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-center text-muted-foreground">
            No components found matching your filters.
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredComponents?.map((component) => (
            <ComponentCard
              key={component.name}
              component={component}
              onSelect={setSelectedComponent}
            />
          ))}
        </div>
      )}

      {/* Detail modal */}
      {selectedComponent && (
        <ComponentDetail name={selectedComponent} onClose={() => setSelectedComponent(null)} />
      )}
    </div>
  )
}

interface ComponentCardProps {
  component: Component
  onSelect: (name: string) => void
}

function ComponentCard({ component, onSelect }: ComponentCardProps) {
  return (
    <Card
      className="cursor-pointer transition-colors hover:bg-secondary/50"
      onClick={() => onSelect(component.name)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onSelect(component.name)}
      aria-label={`View details for ${component.name}`}
    >
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">{component.name}</CardTitle>
          <Badge className={TYPE_COLORS[component.type]}>{component.type}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        {component.description && (
          <p className="mb-3 text-sm text-muted-foreground line-clamp-2">{component.description}</p>
        )}

        {/* Tags */}
        {component.tags && component.tags.length > 0 && (
          <div className="mb-3 flex flex-wrap gap-1">
            {component.tags.slice(0, 3).map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}
            {component.tags.length > 3 && (
              <Badge variant="outline" className="text-xs">
                +{component.tags.length - 3}
              </Badge>
            )}
          </div>
        )}

        {/* Produces/Requires summary */}
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <div className="flex gap-3">
            {component.produces && component.produces.length > 0 && (
              <span>Produces: {component.produces.length}</span>
            )}
            {component.requires && component.requires.length > 0 && (
              <span>Requires: {component.requires.length}</span>
            )}
          </div>
          <ChevronRight className="h-4 w-4" />
        </div>
      </CardContent>
    </Card>
  )
}

interface ComponentDetailProps {
  name: string
  onClose: () => void
}

function ComponentDetail({ name, onClose }: ComponentDetailProps) {
  const { data: component, isLoading } = useComponent(name)

  if (isLoading) {
    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        onClick={onClose}
        role="dialog"
        aria-modal="true"
        aria-label="Loading component details"
      >
        <Card className="w-full max-w-2xl" onClick={(e) => e.stopPropagation()}>
          <CardContent className="flex items-center justify-center p-8">
            <Loader2 className="h-8 w-8 animate-spin" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!component) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="component-detail-title"
    >
      <Card
        className="w-full max-w-2xl max-h-[80vh] overflow-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <CardHeader className="flex flex-row items-center justify-between">
          <div className="flex items-center gap-3">
            <CardTitle id="component-detail-title">{component.name}</CardTitle>
            <Badge className={TYPE_COLORS[component.type]}>{component.type}</Badge>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="Close">
            <X className="h-5 w-5" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          {component.description && (
            <p className="text-muted-foreground">{component.description}</p>
          )}

          {/* Tags */}
          {component.tags && component.tags.length > 0 && (
            <div>
              <h4 className="mb-2 font-semibold">Tags</h4>
              <div className="flex flex-wrap gap-2">
                {component.tags.map((tag) => (
                  <Badge key={tag} variant="secondary">
                    {tag}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {/* Produces */}
          {component.produces && component.produces.length > 0 && (
            <div>
              <h4 className="mb-2 font-semibold">Produces</h4>
              <div className="flex flex-wrap gap-2">
                {component.produces.map((item) => (
                  <Badge key={item} variant="outline" className="text-green-400 border-green-400/50">
                    {item}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {/* Requires */}
          {component.requires && component.requires.length > 0 && (
            <div>
              <h4 className="mb-2 font-semibold">Requires</h4>
              <div className="flex flex-wrap gap-2">
                {component.requires.map((item) => (
                  <Badge key={item} variant="outline" className="text-amber-400 border-amber-400/50">
                    {item}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {/* Source file */}
          <div>
            <h4 className="mb-2 font-semibold">Source File</h4>
            <div className="flex items-center gap-2 rounded-lg border border-border p-3 text-sm">
              <FileCode className="h-4 w-4 text-muted-foreground" />
              <code className="text-muted-foreground">{component.source_file}</code>
            </div>
          </div>

          {/* Scenarios using this component */}
          {component.scenarios && component.scenarios.length > 0 && (
            <div>
              <h4 className="mb-2 font-semibold">Used in Scenarios</h4>
              <div className="space-y-2">
                {component.scenarios.map((scenario) => (
                  <div
                    key={scenario}
                    className="flex items-center gap-2 rounded-lg border border-border p-3 text-sm"
                  >
                    {scenario}
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="flex justify-end pt-4">
            <Button variant="outline" onClick={onClose}>
              Close
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
