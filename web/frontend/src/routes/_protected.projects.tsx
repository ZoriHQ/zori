import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  IconGrid3x3,
  IconList,
  IconPlus,
  IconSearch,
} from '@tabler/icons-react'
import { useProjects } from '@/hooks/use-projects'
import {
  CreateProjectSheet,
  EmptyProjectsState,
  ProjectCard,
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

  const { data: projectsData, isLoading } = useProjects()

  // Use real data if available
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
    console.log('Delete project:', projectId)
    // Call delete API with confirmation
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

  // Show empty state when no projects
  if (!projects || projects.length === 0) {
    return (
      <div className="container mx-auto px-4 py-8">
        <EmptyProjectsState
          onCreateProject={() => setIsCreateSheetOpen(true)}
        />
        <CreateProjectSheet
          open={isCreateSheetOpen}
          onOpenChange={setIsCreateSheetOpen}
        />
      </div>
    )
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

        {/* Stats */}
        <ProjectsStats projects={projects} />

        {/* Filters and Search */}
        <div className="flex flex-col sm:flex-row gap-4 mb-6">
          <div className="relative flex-1">
            <IconSearch className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground h-4 w-4" />
            <Input
              placeholder="Search projects..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-full sm:w-[180px]">
              <SelectValue placeholder="Filter by status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Projects</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="inactive">Inactive</SelectItem>
            </SelectContent>
          </Select>
          <div className="flex gap-2">
            <Button
              variant={view === 'grid' ? 'default' : 'outline'}
              size="icon"
              onClick={() => setView('grid')}
            >
              <IconGrid3x3 className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'list' ? 'default' : 'outline'}
              size="icon"
              onClick={() => setView('list')}
            >
              <IconList className="h-4 w-4" />
            </Button>
          </div>
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
    </div>
  )
}
