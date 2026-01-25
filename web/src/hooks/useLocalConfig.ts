import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchConfig,
  saveConfig,
  validateConfig,
  fetchProject,
  runDiscovery,
  fetchLocalComponents,
  type ChronicleConfig,
  type ValidationResult,
  type ProjectInfo,
  type DiscoveryResult,
} from '@/api/local'

// Project info hook
export function useProject() {
  return useQuery<ProjectInfo>({
    queryKey: ['local', 'project'],
    queryFn: fetchProject,
  })
}

// Config hooks
export function useConfig() {
  return useQuery<ChronicleConfig>({
    queryKey: ['local', 'config'],
    queryFn: fetchConfig,
  })
}

export function useSaveConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: saveConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['local', 'config'] })
      queryClient.invalidateQueries({ queryKey: ['local', 'project'] })
    },
  })
}

export function useValidateConfig() {
  return useMutation<ValidationResult, Error, ChronicleConfig>({
    mutationFn: validateConfig,
  })
}

// Discovery hooks
export function useLocalDiscovery() {
  const queryClient = useQueryClient()
  return useMutation<DiscoveryResult, Error, void>({
    mutationFn: runDiscovery,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['local', 'components'] })
    },
  })
}

export function useLocalComponents() {
  return useQuery<DiscoveryResult>({
    queryKey: ['local', 'components'],
    queryFn: fetchLocalComponents,
  })
}
