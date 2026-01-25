import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { ProjectCard } from './ProjectCard'
import type { Project } from '@/stores/projects'

const mockOnOpen = vi.fn()
const mockOnLaunch = vi.fn()
const mockOnStop = vi.fn()
const mockOnRemove = vi.fn()

const createProject = (overrides: Partial<Project> = {}): Project => ({
  id: 'test-project-1',
  name: 'Test Project',
  path: '/home/user/projects/test',
  addedAt: '2024-01-15T10:00:00Z',
  lastOpened: '2024-01-20T15:30:00Z',
  status: { state: 'stopped' },
  ...overrides,
})

describe('ProjectCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ============================================
  // STATUS COLOR TESTS
  // ============================================
  describe('Status Colors', () => {
    it('shows green indicator for running status', () => {
      const project = createProject({ status: { state: 'running', port: 8080 } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const statusIndicator = screen.getByRole('status')
      expect(statusIndicator).toHaveClass('bg-green-500')
    })

    it('shows gray indicator for stopped status', () => {
      const project = createProject({ status: { state: 'stopped' } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const statusIndicator = screen.getByRole('status')
      expect(statusIndicator).toHaveClass('bg-gray-400')
    })

    it('shows yellow pulsing indicator for starting status', () => {
      const project = createProject({ status: { state: 'starting' } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const statusIndicator = screen.getByRole('status')
      expect(statusIndicator).toHaveClass('bg-yellow-500')
      expect(statusIndicator).toHaveClass('animate-pulse')
    })

    it('shows red indicator for unhealthy status', () => {
      const project = createProject({ status: { state: 'unhealthy', error: 'Connection failed' } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const statusIndicator = screen.getByRole('status')
      expect(statusIndicator).toHaveClass('bg-red-500')
    })

    it('shows light gray indicator for unknown status', () => {
      const project = createProject({ status: { state: 'unknown' } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const statusIndicator = screen.getByRole('status')
      expect(statusIndicator).toHaveClass('bg-gray-300')
    })
  })

  // ============================================
  // BUTTON VISIBILITY TESTS (LOCAL VS REMOTE)
  // ============================================
  describe('Button Visibility - Local vs Remote', () => {
    it('shows Launch button for stopped local project', () => {
      const project = createProject({ status: { state: 'stopped' } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByRole('button', { name: /launch/i })).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /stop/i })).not.toBeInTheDocument()
    })

    it('shows Stop button for running local project', () => {
      const project = createProject({ status: { state: 'running', port: 8080 } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByRole('button', { name: /stop/i })).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /launch/i })).not.toBeInTheDocument()
    })

    it('does not show Launch/Stop buttons for remote project', () => {
      const project = createProject({
        path: undefined,
        remoteUrl: 'https://chronicle.example.com:8080',
        status: { state: 'running', port: 8080 },
      })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.queryByRole('button', { name: /launch/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /stop/i })).not.toBeInTheDocument()
    })

    it('disables Open button for stopped local project', () => {
      const project = createProject({ status: { state: 'stopped' } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const openButton = screen.getByRole('button', { name: /open/i })
      expect(openButton).toBeDisabled()
    })

    it('enables Open button for running local project', () => {
      const project = createProject({ status: { state: 'running', port: 8080 } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const openButton = screen.getByRole('button', { name: /open/i })
      expect(openButton).not.toBeDisabled()
    })

    it('enables Open button for remote project regardless of status', () => {
      const project = createProject({
        path: undefined,
        remoteUrl: 'https://chronicle.example.com:8080',
        status: { state: 'running', port: 8080 },
      })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const openButton = screen.getByRole('button', { name: /open/i })
      expect(openButton).not.toBeDisabled()
    })

    it('does not show Launch/Stop during starting state', () => {
      const project = createProject({ status: { state: 'starting' } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.queryByRole('button', { name: /launch/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /stop/i })).not.toBeInTheDocument()
    })
  })

  // ============================================
  // REMOVE CONFIRMATION TESTS
  // ============================================
  describe('Remove Confirmation', () => {
    it('shows confirmation when remove button is clicked', async () => {
      const user = userEvent.setup()
      const project = createProject()

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const removeButton = screen.getByRole('button', { name: /remove test project/i })
      await user.click(removeButton)

      expect(screen.getByText('Remove project?')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /confirm/i })).toBeInTheDocument()
    })

    it('hides confirmation when cancel is clicked', async () => {
      const user = userEvent.setup()
      const project = createProject()

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const removeButton = screen.getByRole('button', { name: /remove test project/i })
      await user.click(removeButton)

      expect(screen.getByText('Remove project?')).toBeInTheDocument()

      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      await user.click(cancelButton)

      expect(screen.queryByText('Remove project?')).not.toBeInTheDocument()
    })

    it('calls onRemove when confirm is clicked', async () => {
      const user = userEvent.setup()
      const project = createProject()

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const removeButton = screen.getByRole('button', { name: /remove test project/i })
      await user.click(removeButton)

      const confirmButton = screen.getByRole('button', { name: /confirm/i })
      await user.click(confirmButton)

      expect(mockOnRemove).toHaveBeenCalledWith('test-project-1')
    })

    it('does not call onRemove if cancel is clicked', async () => {
      const user = userEvent.setup()
      const project = createProject()

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const removeButton = screen.getByRole('button', { name: /remove test project/i })
      await user.click(removeButton)

      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      await user.click(cancelButton)

      expect(mockOnRemove).not.toHaveBeenCalled()
    })
  })

  // ============================================
  // BUTTON ACTION TESTS
  // ============================================
  describe('Button Actions', () => {
    it('calls onOpen when Open button is clicked', async () => {
      const user = userEvent.setup()
      const project = createProject({ status: { state: 'running', port: 8080 } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const openButton = screen.getByRole('button', { name: /open/i })
      await user.click(openButton)

      expect(mockOnOpen).toHaveBeenCalledWith('test-project-1')
    })

    it('calls onLaunch when Launch button is clicked', async () => {
      const user = userEvent.setup()
      const project = createProject({ status: { state: 'stopped' } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const launchButton = screen.getByRole('button', { name: /launch/i })
      await user.click(launchButton)

      expect(mockOnLaunch).toHaveBeenCalledWith('test-project-1')
    })

    it('calls onStop when Stop button is clicked', async () => {
      const user = userEvent.setup()
      const project = createProject({ status: { state: 'running', port: 8080 } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const stopButton = screen.getByRole('button', { name: /stop/i })
      await user.click(stopButton)

      expect(mockOnStop).toHaveBeenCalledWith('test-project-1')
    })
  })

  // ============================================
  // DISPLAY TESTS
  // ============================================
  describe('Display', () => {
    it('shows project name', () => {
      const project = createProject({ name: 'My Awesome Project' })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByText('My Awesome Project')).toBeInTheDocument()
    })

    it('shows project path for local projects', () => {
      const project = createProject({ path: '/home/user/my-project' })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByText('/home/user/my-project')).toBeInTheDocument()
    })

    it('shows remote URL for remote projects', () => {
      const project = createProject({
        path: undefined,
        remoteUrl: 'https://chronicle.example.com:8080',
      })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByText('https://chronicle.example.com:8080')).toBeInTheDocument()
    })

    it('shows running port when project is running', () => {
      const project = createProject({ status: { state: 'running', port: 9090 } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByText('Running on :9090')).toBeInTheDocument()
    })

    it('shows error message when project has error', () => {
      const project = createProject({
        status: { state: 'unhealthy', error: 'Connection timeout' },
      })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByText('Connection timeout')).toBeInTheDocument()
    })

    it('shows local icon for local projects', () => {
      const project = createProject()

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByTitle('Local project')).toBeInTheDocument()
    })

    it('shows remote icon for remote projects', () => {
      const project = createProject({
        path: undefined,
        remoteUrl: 'https://chronicle.example.com:8080',
      })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByTitle('Remote daemon')).toBeInTheDocument()
    })
  })

  // ============================================
  // DISABLED STATE TESTS
  // ============================================
  describe('Disabled State', () => {
    it('disables all buttons when disabled prop is true', () => {
      const project = createProject({ status: { state: 'running', port: 8080 } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
          disabled={true}
        />
      )

      const buttons = screen.getAllByRole('button')
      buttons.forEach((button) => {
        expect(button).toBeDisabled()
      })
    })
  })

  // ============================================
  // ACCESSIBILITY TESTS
  // ============================================
  describe('Accessibility', () => {
    it('has accessible status indicator with aria-label', () => {
      const project = createProject({ status: { state: 'running', port: 8080 } })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      const statusIndicator = screen.getByRole('status')
      expect(statusIndicator).toHaveAttribute('aria-label', 'Status: Running')
    })

    it('has accessible remove button with project name', () => {
      const project = createProject({ name: 'My Project' })

      render(
        <ProjectCard
          project={project}
          onOpen={mockOnOpen}
          onLaunch={mockOnLaunch}
          onStop={mockOnStop}
          onRemove={mockOnRemove}
        />
      )

      expect(screen.getByRole('button', { name: /remove my project/i })).toBeInTheDocument()
    })
  })
})
