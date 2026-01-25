import { useState, useEffect } from 'react'
import { useConfig, useSaveConfig, useValidateConfig } from '@/hooks/useLocalConfig'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, Save, CheckCircle2, AlertCircle } from 'lucide-react'
import type { ChronicleConfig } from '@/api/local'

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
  const saveConfig = useSaveConfig()
  const validateConfig = useValidateConfig()
  const [activeTab, setActiveTab] = useState<TabId>('general')
  const [editedConfig, setEditedConfig] = useState<ChronicleConfig | null>(null)
  const [validationResult, setValidationResult] = useState<{ valid: boolean; errors: string[] } | null>(null)

  // Initialize edited config when loaded
  useEffect(() => {
    if (config && !editedConfig) {
      setEditedConfig(config)
    }
  }, [config, editedConfig])

  const currentConfig = editedConfig ?? config

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
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

  if (!currentConfig) {
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

  const handleSave = async () => {
    if (!editedConfig) return

    const result = await validateConfig.mutateAsync(editedConfig)
    setValidationResult(result)

    if (result.valid) {
      await saveConfig.mutateAsync(editedConfig)
    }
  }

  const hasChanges = editedConfig !== null && JSON.stringify(editedConfig) !== JSON.stringify(config)

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Configuration</h1>
          <p className="text-muted-foreground">Edit your Chronicle configuration</p>
        </div>
        <div className="flex items-center gap-2">
          {hasChanges && <Badge variant="secondary">Unsaved changes</Badge>}
          {validationResult && !validationResult.valid && (
            <Badge variant="destructive">{validationResult.errors.length} error(s)</Badge>
          )}
          <Button onClick={handleSave} disabled={!hasChanges || saveConfig.isPending}>
            {saveConfig.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : saveConfig.isSuccess ? (
              <CheckCircle2 className="mr-2 h-4 w-4" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            Save
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
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
      <Card>
        <CardHeader>
          <CardTitle>{TABS.find((t) => t.id === activeTab)?.label}</CardTitle>
        </CardHeader>
        <CardContent>
          {activeTab === 'general' && (
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium">Version</label>
                <p className="text-muted-foreground">{currentConfig.version}</p>
              </div>
            </div>
          )}
          {activeTab === 'scenarios' && (
            <div className="space-y-4">
              {currentConfig.scenarios?.map((scenario, i) => (
                <Card key={i}>
                  <CardHeader className="py-3">
                    <CardTitle className="text-base">{scenario.name}</CardTitle>
                  </CardHeader>
                  <CardContent className="py-3">
                    <p className="text-sm text-muted-foreground">
                      {scenario.flow?.length || 0} steps
                      {scenario.tags?.length ? ` - ${scenario.tags.join(', ')}` : ''}
                    </p>
                  </CardContent>
                </Card>
              )) || <p className="text-muted-foreground">No scenarios defined</p>}
            </div>
          )}
          {activeTab === 'infrastructure' && (
            <div className="space-y-4">
              {currentConfig.infrastructure?.providers?.map((provider, i) => (
                <Card key={i}>
                  <CardHeader className="py-3">
                    <CardTitle className="text-base">{provider.name}</CardTitle>
                  </CardHeader>
                  <CardContent className="py-3">
                    <p className="text-sm text-muted-foreground">Type: {provider.type}</p>
                  </CardContent>
                </Card>
              )) || <p className="text-muted-foreground">No infrastructure providers configured</p>}
            </div>
          )}
          {activeTab === 'chaos' && (
            <div className="space-y-4">
              {currentConfig.chaos && Object.keys(currentConfig.chaos).length > 0 ? (
                Object.entries(currentConfig.chaos).map(([name, profile]) => (
                  <Card key={name}>
                    <CardHeader className="py-3">
                      <CardTitle className="text-base">{name}</CardTitle>
                    </CardHeader>
                    <CardContent className="py-3">
                      <p className="text-sm text-muted-foreground">
                        {profile.infrastructure?.length || 0} infrastructure rules,{' '}
                        {profile.application?.length || 0} application rules
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
              {currentConfig.mocks && Object.keys(currentConfig.mocks).length > 0 ? (
                Object.entries(currentConfig.mocks).map(([name, profile]) => (
                  <Card key={name}>
                    <CardHeader className="py-3">
                      <CardTitle className="text-base">{name}</CardTitle>
                    </CardHeader>
                    <CardContent className="py-3">
                      <p className="text-sm text-muted-foreground">
                        {profile.injectors?.length || 0} injectors
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

      {/* Validation Errors */}
      {validationResult && !validationResult.valid && (
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">Validation Errors</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="list-disc list-inside space-y-1">
              {validationResult.errors.map((err, i) => (
                <li key={i} className="text-sm text-destructive">{err}</li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
