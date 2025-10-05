import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  IconArchive,
  IconCalendar,
  IconChartLine,
  IconCode,
  IconDots,
  IconEdit,
  IconExternalLink,
  IconPlus,
  IconStar,
  IconTrash,
  IconWorld,
} from '@tabler/icons-react'

interface ProjectCardProps {
  project: {
    id: string
    name: string
    domain?: string
    website_url?: string
    status?: string
    starred?: boolean
    created_at?: string
    createdAt?: string
    visitors?: number
    pageViews?: number
    avgDuration?: string
    bounceRate?: number
    owner?: {
      name: string
      avatar: string
      initials: string
    }
    team?: Array<{
      name: string
      avatar: string
      initials: string
    }>
  }
  onViewAnalytics?: (projectId: string) => void
  onEdit?: (projectId: string) => void
  onGetCode?: (projectId: string) => void
  onArchive?: (projectId: string) => void
  onDelete?: (projectId: string) => void
}

export function ProjectCard({
  project,
  onViewAnalytics,
  onEdit,
  onGetCode,
  onArchive,
  onDelete,
}: ProjectCardProps) {
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

  const websiteUrl = project.domain || project.website_url || ''
  const displayUrl = websiteUrl.replace(/^https?:\/\//, '')

  return (
    <Card className="hover:shadow-lg transition-shadow">
      <CardHeader>
        <div className="flex justify-between items-start">
          <div className="flex-1">
            <CardTitle className="text-lg flex items-center gap-2">
              {project.name}
              {project.starred && (
                <IconStar className="h-4 w-4 fill-yellow-400 text-yellow-400" />
              )}
            </CardTitle>
            <CardDescription className="mt-2 flex items-center gap-2">
              <IconWorld className="h-3 w-3" />
              <a
                href={websiteUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="hover:underline flex items-center gap-1"
              >
                {displayUrl}
                <IconExternalLink className="h-3 w-3" />
              </a>
            </CardDescription>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <IconDots className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => onViewAnalytics?.(project.id)}>
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
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <Badge className={getStatusColor(project.status || 'active')}>
              {project.status || 'active'}
            </Badge>
            {(project.created_at || project.createdAt) && (
              <Badge variant="outline" className="text-xs">
                <IconCalendar className="mr-1 h-3 w-3" />
                Created{' '}
                {new Date(
                  project.created_at || project.createdAt || '',
                ).toLocaleDateString()}
              </Badge>
            )}
          </div>

          <Tabs defaultValue="metrics" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="metrics">Metrics</TabsTrigger>
              <TabsTrigger value="team">Team</TabsTrigger>
            </TabsList>
            <TabsContent value="metrics" className="space-y-3 mt-3">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">Visitors</p>
                  <p className="text-lg font-semibold">
                    {(project.visitors || 0).toLocaleString()}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">Page Views</p>
                  <p className="text-lg font-semibold">
                    {(project.pageViews || 0).toLocaleString()}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">Avg Duration</p>
                  <p className="text-lg font-semibold">
                    {project.avgDuration || 'N/A'}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">Bounce Rate</p>
                  <p className="text-lg font-semibold">
                    {project.bounceRate ? `${project.bounceRate}%` : 'N/A'}
                  </p>
                </div>
              </div>
            </TabsContent>
            <TabsContent value="team" className="mt-3">
              <div className="space-y-3">
                {project.owner && (
                  <div className="flex items-center gap-2">
                    <Avatar className="h-6 w-6">
                      <AvatarImage src={project.owner.avatar} />
                      <AvatarFallback className="text-xs">
                        {project.owner.initials}
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex-1">
                      <p className="text-sm font-medium">{project.owner.name}</p>
                      <p className="text-xs text-muted-foreground">Owner</p>
                    </div>
                  </div>
                )}
                <div className="flex -space-x-2">
                  {(project.team || []).map((member, index) => (
                    <Avatar
                      key={index}
                      className="h-8 w-8 border-2 border-background"
                    >
                      <AvatarImage src={member.avatar} />
                      <AvatarFallback className="text-xs">
                        {member.initials}
                      </AvatarFallback>
                    </Avatar>
                  ))}
                  <Button variant="ghost" size="sm" className="ml-2">
                    <IconPlus className="h-3 w-3 mr-1" />
                    Add
                  </Button>
                </div>
              </div>
            </TabsContent>
          </Tabs>

          <div className="flex gap-2">
            <Button
              variant="default"
              size="sm"
              className="flex-1"
              onClick={() => onViewAnalytics?.(project.id)}
            >
              <IconChartLine className="mr-2 h-4 w-4" />
              View Analytics
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onGetCode?.(project.id)}
            >
              <IconCode className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
