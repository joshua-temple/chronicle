import { apiRequest } from './client'
import type {
  Project,
  ProjectHealth,
  ProjectSettings,
  HealthResponse,
} from './types'

const PROJECTS_STORAGE_KEY = 'chronicle-projects'

/**
 * Project management API
 * Projects are stored locally and connect to remote daemons
 */

// =============================================================================
// Local Project Storage (localStorage)
// =============================================================================

export function getStoredProjects(): Project[] {
  try {
    const stored = localStorage.getItem(PROJECTS_STORAGE_KEY)
    return stored ? JSON.parse(stored) : []
  } catch {
    return []
  }
}

export function saveStoredProjects(projects: Project[]): void {
  localStorage.setItem(PROJECTS_STORAGE_KEY, JSON.stringify(projects))
}

export function addStoredProject(project: Omit<Project, 'id' | 'addedAt' | 'status'>): Project {
  const projects = getStoredProjects()
  const newProject: Project = {
    ...project,
    id: crypto.randomUUID(),
    addedAt: new Date().toISOString(),
    status: 'disconnected',
  }
  projects.push(newProject)
  saveStoredProjects(projects)
  return newProject
}

export function updateStoredProject(id: string, updates: Partial<Project>): Project | null {
  const projects = getStoredProjects()
  const index = projects.findIndex(p => p.id === id)
  if (index === -1) return null

  projects[index] = { ...projects[index], ...updates }
  saveStoredProjects(projects)
  return projects[index]
}

export function removeStoredProject(id: string): boolean {
  const projects = getStoredProjects()
  const filtered = projects.filter(p => p.id !== id)
  if (filtered.length === projects.length) return false

  saveStoredProjects(filtered)
  return true
}

// =============================================================================
// Daemon Connection
// =============================================================================

export async function checkDaemonHealth(daemonUrl: string): Promise<ProjectHealth | null> {
  try {
    const response = await fetch(`${daemonUrl}/api/v1/health`, {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
      signal: AbortSignal.timeout(5000),
    })

    if (!response.ok) {
      return null
    }

    const data = await response.json()
    return {
      projectId: '', // Will be set by caller
      status: 'connected',
      version: data.version,
      uptime: data.uptime,
      activeRuns: data.active_runs || 0,
    }
  } catch {
    return null
  }
}

export async function connectToProject(project: Project): Promise<ProjectHealth> {
  const health = await checkDaemonHealth(project.daemonUrl)

  if (health) {
    return {
      ...health,
      projectId: project.id,
    }
  }

  return {
    projectId: project.id,
    status: 'error',
    activeRuns: 0,
    error: 'Failed to connect to daemon',
  }
}

// =============================================================================
// Auto-Discovery
// =============================================================================

export async function discoverProjects(): Promise<Project[]> {
  // Try common local ports for Chronicle daemons
  const commonPorts = [3000, 8080, 8000, 9000]
  const discovered: Project[] = []

  const checks = commonPorts.map(async (port) => {
    const url = `http://localhost:${port}`
    const health = await checkDaemonHealth(url)

    if (health) {
      // Check if we already have this project
      const existing = getStoredProjects()
      const alreadyAdded = existing.some(p => p.daemonUrl === url)

      if (!alreadyAdded) {
        discovered.push({
          id: crypto.randomUUID(),
          name: `Chronicle (port ${port})`,
          daemonUrl: url,
          status: 'connected',
          addedAt: new Date().toISOString(),
          autoDiscovered: true,
        })
      }
    }
  })

  await Promise.allSettled(checks)
  return discovered
}

// =============================================================================
// Project Settings (stored on daemon)
// =============================================================================

export async function fetchProjectSettings(
  daemonUrl: string
): Promise<ProjectSettings> {
  return apiRequest<ProjectSettings>(`${daemonUrl}/api/v1/settings`)
}

export async function updateProjectSettings(
  daemonUrl: string,
  settings: Partial<ProjectSettings>
): Promise<ProjectSettings> {
  return apiRequest<ProjectSettings>(`${daemonUrl}/api/v1/settings`, {
    method: 'PUT',
    body: JSON.stringify(settings),
  })
}

// =============================================================================
// Aggregate Health Check
// =============================================================================

export async function fetchAggregateHealth(projects: Project[]): Promise<HealthResponse> {
  const healthChecks = projects.map(async (project) => {
    return connectToProject(project)
  })

  const results = await Promise.allSettled(healthChecks)
  const projectHealths: ProjectHealth[] = results.map((result, index) => {
    if (result.status === 'fulfilled') {
      return result.value
    }
    return {
      projectId: projects[index].id,
      status: 'error' as const,
      activeRuns: 0,
      error: 'Connection check failed',
    }
  })

  // Determine aggregate status
  const connected = projectHealths.filter(h => h.status === 'connected').length
  const total = projectHealths.length

  let aggregateStatus: 'healthy' | 'degraded' | 'unhealthy'
  if (connected === total) {
    aggregateStatus = 'healthy'
  } else if (connected > 0) {
    aggregateStatus = 'degraded'
  } else {
    aggregateStatus = 'unhealthy'
  }

  return {
    projects: projectHealths,
    aggregateStatus,
  }
}
