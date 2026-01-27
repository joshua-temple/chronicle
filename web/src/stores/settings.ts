import { create } from 'zustand'
import { getUISettings, saveUISettings, resetUISettings } from '@/api/settings'
import type { UISettings } from '@/api/types'

interface SettingsState {
  settings: UISettings
  loading: boolean

  // Actions
  loadSettings: () => void
  updateSettings: (updates: Partial<UISettings>) => void
  resetSettings: () => void

  // Convenience setters
  setTheme: (theme: UISettings['theme']) => void
  toggleSidebar: () => void
  setDefaultView: (view: UISettings['defaultView']) => void
  setRefreshInterval: (interval: number) => void
  setNotification: (key: keyof UISettings['notifications'], value: boolean) => void
}

export const useSettingsStore = create<SettingsState>((set, get) => ({
  settings: getUISettings(),
  loading: false,

  loadSettings: () => {
    const settings = getUISettings()
    set({ settings })
  },

  updateSettings: (updates) => {
    const settings = saveUISettings(updates)
    set({ settings })
  },

  resetSettings: () => {
    const settings = resetUISettings()
    set({ settings })
  },

  setTheme: (theme) => {
    get().updateSettings({ theme })
  },

  toggleSidebar: () => {
    const { settings, updateSettings } = get()
    updateSettings({ sidebarCollapsed: !settings.sidebarCollapsed })
  },

  setDefaultView: (defaultView) => {
    get().updateSettings({ defaultView })
  },

  setRefreshInterval: (refreshInterval) => {
    get().updateSettings({ refreshInterval })
  },

  setNotification: (key, value) => {
    const { settings, updateSettings } = get()
    updateSettings({
      notifications: { ...settings.notifications, [key]: value },
    })
  },
}))

// Convenience hooks
export function useTheme() {
  return useSettingsStore(state => state.settings.theme)
}

export function useSidebarCollapsed() {
  return useSettingsStore(state => state.settings.sidebarCollapsed)
}

export function useRefreshInterval() {
  return useSettingsStore(state => state.settings.refreshInterval)
}

export function useNotificationSettings() {
  return useSettingsStore(state => state.settings.notifications)
}
