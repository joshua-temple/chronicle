import { useState } from 'react'
import { useConfig } from '@/hooks/useLocalConfig'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle } from 'lucide-react'

type TabId = 'general' | 'scenarios' | 'infrastructure' | 'chaos' | 'mocks'

const TABS: { id: TabId; label: string }[] = [
  { id: 'general', label: 'General' },
  { id: 'scenarios', label: 'Scenarios' },
  { id: 'infrastructure', label: 'Infrastructure' },
  { id: 'chaos', label: 'Chaos' },
  { id: 'mocks', label: 'Mocks' },
]

export function ConfigEditor() {
  const { data: config, isLoading, error } = useConfig()
  const [activeTab, setActiveTab] = useState<TabId>('general')

  if (isLoading) {
    return (
      <div className="p-6 space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-64 mt-2" />
          </div>
          <Skeleton className="h-10 w-20" />
        </div>
        <div className="flex gap-1 border-b pb-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} className="h-8 w-24" />
          ))}
        </div>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-32" />
          </CardHeader>
          <CardContent className="space-y-4">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center text-destructive">
              <AlertCircle className="mx-auto h-8 w-8 mb-2" />
              <p>Failed to load configuration</p>
              <p className="text-sm text-muted-foreground mt-1">{error.message}</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!config) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="pt-6 text-center text-muted-foreground">
            No configuration found. Create a chronicle.yaml file to get started.
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Configuration Viewer</h1>
          <p className="text-muted-foreground">
            View your Chronicle configuration &middot; Edit <code className="text-xs bg-muted px-1 py-0.5 rounded">chronicle.yaml</code> directly
          </p>
        </div>
        <Badge variant="outline" className="text-muted-foreground">
          Read-only
        </Badge>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b" role="tablist" aria-label="Configuration sections">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            id={`tab-${tab.id}`}
            role="tab"
            aria-selected={activeTab === tab.id}
            aria-controls={`panel-${tab.id}`}
            tabIndex={activeTab === tab.id ? 0 : -1}
            onClick={() => setActiveTab(tab.id)}
            onKeyDown={(e) => {
              const tabIds = TABS.map(t => t.id)
              const currentIndex = tabIds.indexOf(activeTab)
              if (e.key === 'ArrowRight') {
                const nextIndex = (currentIndex + 1) % tabIds.length
                setActiveTab(tabIds[nextIndex])
              } else if (e.key === 'ArrowLeft') {
                const prevIndex = (currentIndex - 1 + tabIds.length) % tabIds.length
                setActiveTab(tabIds[prevIndex])
              } else if (e.key === 'Home') {
                setActiveTab(tabIds[0])
              } else if (e.key === 'End') {
                setActiveTab(tabIds[tabIds.length - 1])
              }
            }}
            className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
              activeTab === tab.id
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <Card
        role="tabpanel"
        id={`panel-${activeTab}`}
        aria-labelledby={`tab-${activeTab}`}
        tabIndex={0}
      >
        <CardHeader>
          <CardTitle>{TABS.find((t) => t.id === activeTab)?.label}</CardTitle>
        </CardHeader>
        <CardContent>
          {activeTab === 'general' && (
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium">Version</label>
                <p className="text-muted-foreground">{config.version}</p>
              </div>
            </div>
          )}
          {activeTab === 'scenarios' && (
            <div className="space-y-4">
              {config.scenarios?.map((scenario, i) => (
                <Card key={i}>
                  <CardHeader className="py-3">
                    <CardTitle className="text-base">{scenario.name}</CardTitle>
                  </CardHeader>
                  <CardContent className="py-3">
                    <p className="text-sm text-muted-foreground">
                      {scenario.flow?.length || 0} {(scenario.flow?.length || 0) === 1 ? 'step' : 'steps'}
                      {scenario.tags?.length ? ` - ${scenario.tags.join(', ')}` : ''}
                    </p>
                  </CardContent>
                </Card>
              )) || <p className="text-muted-foreground">No scenarios defined</p>}
            </div>
          )}
          {activeTab === 'infrastructure' && (
            <div className="space-y-4">
              {config.infrastructure && Object.keys(config.infrastructure).length > 0 ? (
                Object.entries(config.infrastructure).map(([name, config]) => (
                  <Card key={name}>
                    <CardHeader className="py-3">
                      <CardTitle className="text-base">{name}</CardTitle>
                    </CardHeader>
                    <CardContent className="py-3">
                      <p className="text-sm text-muted-foreground">
                        {typeof config === 'object' ? JSON.stringify(config).slice(0, 100) : String(config)}
                      </p>
                    </CardContent>
                  </Card>
                ))
              ) : (
                <p className="text-muted-foreground">No infrastructure providers configured</p>
              )}
            </div>
          )}
          {activeTab === 'chaos' && (
            <div className="space-y-4">
              {config.chaos_profiles && Object.keys(config.chaos_profiles).length > 0 ? (
                Object.entries(config.chaos_profiles).map(([name, profile]) => (
                  <Card key={name}>
                    <CardHeader className="py-3">
                      <CardTitle className="text-base">{name}</CardTitle>
                    </CardHeader>
                    <CardContent className="py-3">
                      <p className="text-sm text-muted-foreground">
                        {profile.name || 'No description'}
                      </p>
                    </CardContent>
                  </Card>
                ))
              ) : (
                <p className="text-muted-foreground">No chaos profiles configured</p>
              )}
            </div>
          )}
          {activeTab === 'mocks' && (
            <div className="space-y-4">
              {config.mock_profiles && Object.keys(config.mock_profiles).length > 0 ? (
                Object.entries(config.mock_profiles).map(([name, profile]) => (
                  <Card key={name}>
                    <CardHeader className="py-3">
                      <CardTitle className="text-base">{name}</CardTitle>
                    </CardHeader>
                    <CardContent className="py-3">
                      <p className="text-sm text-muted-foreground">
                        {profile.name || 'No description'}
                      </p>
                    </CardContent>
                  </Card>
                ))
              ) : (
                <p className="text-muted-foreground">No mock profiles configured</p>
              )}
            </div>
          )}
        </CardContent>
      </Card>

    </div>
  )
}
