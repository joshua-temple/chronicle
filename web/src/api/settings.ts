import type { UISettings } from './types'

const UI_SETTINGS_KEY = 'chronicle-ui-settings'

const DEFAULT_UI_SETTINGS: UISettings = {
  theme: 'system',
  sidebarCollapsed: false,
  defaultView: 'dashboard',
  refreshInterval: 5000,
  notifications: {
    runCompleted: true,
    runFailed: true,
    connectionLost: true,
  },
}

export function getUISettings(): UISettings {
  try {
    const stored = localStorage.getItem(UI_SETTINGS_KEY)
    if (!stored) return DEFAULT_UI_SETTINGS
    return { ...DEFAULT_UI_SETTINGS, ...JSON.parse(stored) }
  } catch {
    return DEFAULT_UI_SETTINGS
  }
}

export function saveUISettings(settings: Partial<UISettings>): UISettings {
  const current = getUISettings()
  const updated = { ...current, ...settings }
  localStorage.setItem(UI_SETTINGS_KEY, JSON.stringify(updated))
  return updated
}

export function resetUISettings(): UISettings {
  localStorage.removeItem(UI_SETTINGS_KEY)
  return DEFAULT_UI_SETTINGS
}
