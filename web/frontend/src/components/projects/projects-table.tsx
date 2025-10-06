import {
  IconArchive,
  IconChartLine,
  IconCode,
  IconDots,
  IconEdit,
  IconStar,
  IconTrash,
  IconWorld,
} from '@tabler/icons-react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface ProjectsTableProps {
  projects: Array<{
    id: string
    name: string
    domain?: string
    website_url?: string
    status?: string
    starred?: boolean
    visitors?: number
    pageViews?: number
    avgDuration?: string
    bounceRate?: number
    team?: Array<{
      name: string
      avatar: string
      initials: string
    }>
  }>
  selectedProjects: Array<string>
  onSelectProject: (projectId: string, checked: boolean) => void
  onSelectAll: (checked: boolean) => void
  onViewAnalytics?: (projectId: string) => void
  onEdit?: (projectId: string) => void
  onGetCode?: (projectId: string) => void
  onArchive?: (projectId: string) => void
  onDelete?: (projectId: string) => void
}

export function ProjectsTable({
  projects,
  selectedProjects,
  onSelectProject,
  onSelectAll,
  onViewAnalytics,
  onEdit,
  onGetCode,
  onArchive,
  onDelete,
}: ProjectsTableProps) {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
      case 'inactive':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  return (
    <Card>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-12">
              <Checkbox
                checked={
                  projects.length > 0 &&
                  selectedProjects.length === projects.length
                }
                onCheckedChange={onSelectAll}
              />
            </TableHead>
            <TableHead>Project</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Visitors</TableHead>
            <TableHead>Page Views</TableHead>
            <TableHead>Avg Duration</TableHead>
            <TableHead>Bounce Rate</TableHead>
            <TableHead>Team</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {projects.map((project) => {
            const websiteUrl = project.domain || project.website_url || ''
            const displayUrl = websiteUrl.replace(/^https?:\/\//, '')

            return (
              <TableRow key={project.id}>
                <TableCell>
                  <Checkbox
                    checked={selectedProjects.includes(project.id)}
                    onCheckedChange={(checked) =>
                      onSelectProject(project.id, checked as boolean)
                    }
                  />
                </TableCell>
                <TableCell>
                  <div>
                    <div className="font-medium flex items-center gap-2">
                      {project.name}
                      {project.starred && (
                        <IconStar className="h-3 w-3 fill-yellow-400 text-yellow-400" />
                      )}
                    </div>
                    <div className="text-sm text-muted-foreground flex items-center gap-1">
                      <IconWorld className="h-3 w-3" />
                      <a
                        href={websiteUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="hover:underline"
                      >
                        {displayUrl}
                      </a>
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge className={getStatusColor(project.status || 'active')}>
                    {project.status || 'active'}
                  </Badge>
                </TableCell>
                <TableCell>
                  {(project.visitors || 0).toLocaleString()}
                </TableCell>
                <TableCell>
                  {(project.pageViews || 0).toLocaleString()}
                </TableCell>
                <TableCell>{project.avgDuration || 'N/A'}</TableCell>
                <TableCell>
                  {project.bounceRate ? `${project.bounceRate}%` : 'N/A'}
                </TableCell>
                <TableCell>
                  <div className="flex -space-x-1">
                    {(project.team || []).slice(0, 3).map((member, index) => (
                      <Avatar
                        key={index}
                        className="h-6 w-6 border border-background"
                      >
                        <AvatarImage src={member.avatar} />
                        <AvatarFallback className="text-xs">
                          {member.initials}
                        </AvatarFallback>
                      </Avatar>
                    ))}
                    {project.team && project.team.length > 3 && (
                      <div className="h-6 w-6 rounded-full bg-muted border border-background flex items-center justify-center">
                        <span className="text-xs">
                          +{project.team.length - 3}
                        </span>
                      </div>
                    )}
                  </div>
                </TableCell>
                <TableCell className="text-right">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-8 w-8">
                        <IconDots className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuLabel>Actions</DropdownMenuLabel>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        onClick={() => onViewAnalytics?.(project.id)}
                      >
                        <IconChartLine className="mr-2 h-4 w-4" />
                        View Analytics
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => onEdit?.(project.id)}>
                        <IconEdit className="mr-2 h-4 w-4" />
                        Edit Settings
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => onGetCode?.(project.id)}>
                        <IconCode className="mr-2 h-4 w-4" />
                        Get Tracking Code
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => onArchive?.(project.id)}>
                        <IconArchive className="mr-2 h-4 w-4" />
                        Archive
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={() => onDelete?.(project.id)}
                      >
                        <IconTrash className="mr-2 h-4 w-4" />
                        Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </Card>
  )
}
