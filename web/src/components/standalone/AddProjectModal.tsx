import { useState, useCallback, useEffect } from 'react'
import { Modal } from '@/components/ui/modal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Folder, Globe } from 'lucide-react'
import { cn } from '@/lib/utils'

type ProjectType = 'local' | 'remote'

interface AddProjectModalProps {
  open: boolean
  onClose: () => void
  onSubmit: (project: { name: string; path?: string; remoteUrl?: string }) => void
  loading?: boolean
}

function extractProjectName(path: string): string {
  if (!path) return ''
  // Handle both forward and backslashes, remove trailing slashes
  const normalized = path.replace(/\\/g, '/').replace(/\/+$/, '')
  const parts = normalized.split('/')
  return parts[parts.length - 1] || ''
}

function validatePath(path: string): string | null {
  if (!path.trim()) {
    return 'Path is required'
  }
  // Basic path validation - should look like an absolute path
  const trimmed = path.trim()
  if (!trimmed.startsWith('/') && !trimmed.match(/^[A-Za-z]:\\/)) {
    return 'Please enter an absolute path'
  }
  return null
}

function validateUrl(url: string): string | null {
  if (!url.trim()) {
    return 'URL is required'
  }
  try {
    const parsed = new URL(url.trim())
    if (!['http:', 'https:'].includes(parsed.protocol)) {
      return 'URL must use http or https protocol'
    }
    return null
  } catch {
    return 'Please enter a valid URL'
  }
}

export function AddProjectModal({ open, onClose, onSubmit, loading = false }: AddProjectModalProps) {
  const [projectType, setProjectType] = useState<ProjectType>('local')
  const [name, setName] = useState('')
  const [path, setPath] = useState('')
  const [remoteUrl, setRemoteUrl] = useState('')
  const [errors, setErrors] = useState<{ name?: string; path?: string; remoteUrl?: string }>({})
  const [touched, setTouched] = useState<{ name?: boolean; path?: boolean; remoteUrl?: boolean }>({})

  // Reset form when modal opens/closes
  useEffect(() => {
    if (!open) {
      setProjectType('local')
      setName('')
      setPath('')
      setRemoteUrl('')
      setErrors({})
      setTouched({})
    }
  }, [open])

  // Auto-detect name from path
  useEffect(() => {
    if (projectType === 'local' && path && !touched.name) {
      const detectedName = extractProjectName(path)
      if (detectedName) {
        setName(detectedName)
      }
    }
  }, [path, projectType, touched.name])

  const validate = useCallback((): boolean => {
    const newErrors: typeof errors = {}

    if (!name.trim()) {
      newErrors.name = 'Name is required'
    }

    if (projectType === 'local') {
      const pathError = validatePath(path)
      if (pathError) newErrors.path = pathError
    } else {
      const urlError = validateUrl(remoteUrl)
      if (urlError) newErrors.remoteUrl = urlError
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }, [name, path, remoteUrl, projectType])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    // Mark all fields as touched
    setTouched({ name: true, path: true, remoteUrl: true })

    if (!validate()) return

    onSubmit({
      name: name.trim(),
      path: projectType === 'local' ? path.trim() : undefined,
      remoteUrl: projectType === 'remote' ? remoteUrl.trim() : undefined,
    })
  }

  const handleClose = () => {
    if (!loading) {
      onClose()
    }
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Add Project"
      titleId="add-project-modal-title"
      className="max-w-md"
    >
      <form onSubmit={handleSubmit} className="px-6 pb-6">
        {/* Project Type Selection */}
        <div className="mb-6">
          <label className="block text-sm font-medium mb-3">Project Type</label>
          <div className="grid grid-cols-2 gap-3">
            <button
              type="button"
              onClick={() => setProjectType('local')}
              className={cn(
                'flex items-center justify-center gap-2 p-4 rounded-lg border-2 transition-colors',
                projectType === 'local'
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border hover:border-primary/50'
              )}
            >
              <Folder className="h-5 w-5" />
              <span className="font-medium">Local Project</span>
            </button>
            <button
              type="button"
              onClick={() => setProjectType('remote')}
              className={cn(
                'flex items-center justify-center gap-2 p-4 rounded-lg border-2 transition-colors',
                projectType === 'remote'
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border hover:border-primary/50'
              )}
            >
              <Globe className="h-5 w-5" />
              <span className="font-medium">Remote Daemon</span>
            </button>
          </div>
        </div>

        {/* Path or URL Input */}
        <div className="mb-4">
          {projectType === 'local' ? (
            <>
              <label htmlFor="project-path" className="block text-sm font-medium mb-2">
                Project Path
              </label>
              <Input
                id="project-path"
                type="text"
                placeholder="/path/to/your/project"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                onBlur={() => setTouched((t) => ({ ...t, path: true }))}
                disabled={loading}
                aria-describedby={errors.path ? 'path-error' : undefined}
                aria-invalid={errors.path && touched.path ? true : undefined}
                className={cn(errors.path && touched.path && 'border-destructive')}
              />
              {errors.path && touched.path && (
                <p id="path-error" className="mt-1 text-sm text-destructive">
                  {errors.path}
                </p>
              )}
              <p className="mt-1 text-xs text-muted-foreground">
                Enter the absolute path to your Chronicle project
              </p>
            </>
          ) : (
            <>
              <label htmlFor="remote-url" className="block text-sm font-medium mb-2">
                Daemon URL
              </label>
              <Input
                id="remote-url"
                type="url"
                placeholder="https://chronicle.example.com:8080"
                value={remoteUrl}
                onChange={(e) => setRemoteUrl(e.target.value)}
                onBlur={() => setTouched((t) => ({ ...t, remoteUrl: true }))}
                disabled={loading}
                aria-describedby={errors.remoteUrl ? 'url-error' : undefined}
                aria-invalid={errors.remoteUrl && touched.remoteUrl ? true : undefined}
                className={cn(errors.remoteUrl && touched.remoteUrl && 'border-destructive')}
              />
              {errors.remoteUrl && touched.remoteUrl && (
                <p id="url-error" className="mt-1 text-sm text-destructive">
                  {errors.remoteUrl}
                </p>
              )}
              <p className="mt-1 text-xs text-muted-foreground">
                Enter the URL of a running Chronicle daemon
              </p>
            </>
          )}
        </div>

        {/* Name Input */}
        <div className="mb-6">
          <label htmlFor="project-name" className="block text-sm font-medium mb-2">
            Display Name
          </label>
          <Input
            id="project-name"
            type="text"
            placeholder="my-service"
            value={name}
            onChange={(e) => {
              setName(e.target.value)
              setTouched((t) => ({ ...t, name: true }))
            }}
            onBlur={() => setTouched((t) => ({ ...t, name: true }))}
            disabled={loading}
            aria-describedby={errors.name ? 'name-error' : undefined}
            aria-invalid={errors.name && touched.name ? true : undefined}
            className={cn(errors.name && touched.name && 'border-destructive')}
          />
          {errors.name && touched.name && (
            <p id="name-error" className="mt-1 text-sm text-destructive">
              {errors.name}
            </p>
          )}
          {projectType === 'local' && (
            <p className="mt-1 text-xs text-muted-foreground">
              Auto-detected from path. You can customize it.
            </p>
          )}
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <Button
            type="button"
            variant="outline"
            onClick={handleClose}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={loading}>
            {loading ? 'Adding...' : 'Add Project'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
