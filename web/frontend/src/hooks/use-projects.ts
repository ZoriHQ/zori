import { useMutation, useQuery } from '@tanstack/react-query'
import type Zoriapi from 'zorihq'
import { useApiClient } from '@/lib/api-client'

export function useProjects() {
  const zClient = useApiClient()

  return useQuery<Zoriapi.V1.Projects.ListProjectsResponse>({
    queryKey: ['projects'],
    queryFn: () => zClient.v1.projects.list(),
  })
}

export function useProject(projectId: string) {
  const zClient = useApiClient()

  return useQuery<Zoriapi.V1.Projects.Project>({
    queryKey: ['projects', projectId],
    queryFn: () => zClient.v1.projects.get(projectId),
    enabled: !!projectId,
  })
}

export function useCreateProject() {
  const zClient = useApiClient()

  return useMutation<
    Zoriapi.V1.Projects.ProjectResponse,
    Error,
    Zoriapi.V1.Projects.ProjectCreateParams
  >({
    mutationFn: (data) => zClient.v1.projects.create(data),
  })
}

export function useUpdateProject(projectId: string) {
  const zClient = useApiClient()

  return useMutation<
    Zoriapi.V1.Projects.ProjectResponse,
    Error,
    Zoriapi.V1.Projects.ProjectCreateParams
  >({
    mutationFn: (data) => zClient.v1.projects.update(projectId, data),
  })
}

export function useDeleteProject(projectId: string) {
  const zClient = useApiClient()

  return useMutation<Zoriapi.V1.Projects.ProjectDeleteResponse, Error>({
    mutationFn: () => zClient.v1.projects.delete(projectId),
  })
}
