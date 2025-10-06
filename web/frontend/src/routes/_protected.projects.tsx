import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import {
  IconGrid3x3,
  IconList,
  IconPlus,
  IconSearch,
} from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useDeleteProject, useProjects } from '@/hooks/use-projects'
import {
  CreateProjectSheet,
  DeleteProjectDialog,
  EmptyProjectsState,
  ProjectCard,
  ProjectOnboarding,
  ProjectsStats,
  ProjectsTable,
} from '@/components/projects'

export const Route = createFileRoute('/_protected/projects')({
  component: ProjectsPage,
})

function ProjectsPage() {
  const [view, setView] = useState<'grid' | 'list'>('grid')
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [selectedProjects, setSelectedProjects] = useState<Array<string>>([])
  const [isCreateSheetOpen, setIsCreateSheetOpen] = useState(false)
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean
    projectId: string
    projectName: string
  }>({
    open: false,
    projectId: '',
    projectName: '',
  })

  const { data: projectsData, isLoading } = useProjects()
  const queryClient = useQueryClient()
  const deleteProjectMutation = useDeleteProject()

  const projects = projectsData?.projects || []

  const filteredProjects = projects.filter((project: any) => {
    const matchesSearch =
      project.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (project.domain || project.website_url || '')
        ?.toLowerCase()
        .includes(searchQuery.toLowerCase())
    const matchesStatus =
      statusFilter === 'all' || project.status === statusFilter
    return matchesSearch && matchesStatus
  })

  const handleSelectProject = (projectId: string, checked: boolean) => {
    if (checked) {
      setSelectedProjects([...selectedProjects, projectId])
    } else {
      setSelectedProjects(selectedProjects.filter((id) => id !== projectId))
    }
  }

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedProjects(filteredProjects.map((p: any) => p.id))
    } else {
      setSelectedProjects([])
    }
  }

  // Action handlers (to be implemented with actual functionality)
  const handleViewAnalytics = (projectId: string) => {
    console.log('View analytics for:', projectId)
    // Navigate to analytics page
  }

  const handleEdit = (projectId: string) => {
    console.log('Edit project:', projectId)
    // Open edit modal or navigate to edit page
  }

  const handleGetCode = (projectId: string) => {
    console.log('Get tracking code for:', projectId)
    // Open modal with tracking code
  }

  const handleArchive = (projectId: string) => {
    console.log('Archive project:', projectId)
    // Call archive API
  }

  const handleDelete = (projectId: string) => {
    const project = projects.find((p: any) => p.id === projectId)
    if (project) {
      setDeleteDialog({
        open: true,
        projectId,
        projectName: project.name ?? '',
      })
    }
  }

  const confirmDelete = async () => {
    try {
      await deleteProjectMutation.mutateAsync({
        projectId: deleteDialog.projectId,
      })

      await queryClient.invalidateQueries({ queryKey: ['projects'] })

      toast.success(`${deleteDialog.projectName} has been permanently deleted.`)

      setDeleteDialog({ open: false, projectId: '', projectName: '' })
    } catch (error) {
      toast.error('Failed to delete the project. Please try again.')
    }
  }

  // Show loading state
  if (isLoading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="flex items-center justify-center h-64">
          <div className="text-muted-foreground">Loading projects...</div>
        </div>
      </div>
    )
  }

  if (projects.length === 0) {
    return <ProjectOnboarding />
  }

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <div className="flex justify-between items-start mb-4">
          <div>
            <h1 className="text-3xl font-bold mb-2">Analytics Projects</h1>
            <p className="text-muted-foreground">
              Track and analyze your website performance with Zori Analytics
            </p>
          </div>
          <CreateProjectSheet
            open={isCreateSheetOpen}
            onOpenChange={setIsCreateSheetOpen}
            trigger={
              <Button>
                <IconPlus className="mr-2 h-4 w-4" />
                New Project
              </Button>
            }
          />
        </div>
      </div>

      {/* Projects View */}
      {view === 'grid' ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredProjects.map((project: any) => (
            <ProjectCard
              key={project.id}
              project={project}
              onViewAnalytics={handleViewAnalytics}
              onEdit={handleEdit}
              onGetCode={handleGetCode}
              onArchive={handleArchive}
              onDelete={handleDelete}
            />
          ))}
        </div>
      ) : (
        <ProjectsTable
          projects={filteredProjects}
          selectedProjects={selectedProjects}
          onSelectProject={handleSelectProject}
          onSelectAll={handleSelectAll}
          onViewAnalytics={handleViewAnalytics}
          onEdit={handleEdit}
          onGetCode={handleGetCode}
          onArchive={handleArchive}
          onDelete={handleDelete}
        />
      )}

      <DeleteProjectDialog
        open={deleteDialog.open}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteDialog({ open: false, projectId: '', projectName: '' })
          }
        }}
        projectName={deleteDialog.projectName}
        isDeleting={deleteProjectMutation.isPending}
        onConfirm={confirmDelete}
      />
    </div>
  )
}
